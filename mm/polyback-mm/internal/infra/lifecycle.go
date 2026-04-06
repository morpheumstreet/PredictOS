package infra

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
)

type Manager struct {
	cfg    *config.Root
	home   string
	mu     sync.RWMutex
	running bool
}

func NewManager(cfg *config.Root) (*Manager, error) {
	home := cfg.ResolvePolybotHome()
	if home == "" {
		return nil, fmt.Errorf("POLYBOT_HOME or infrastructure.polybot_home must point to polybot-main (for docker compose files)")
	}
	return &Manager{cfg: cfg, home: home}, nil
}

func (m *Manager) PolybotHome() string { return m.home }

func (m *Manager) StartAll() error {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	stacks := append([]config.InfraStack(nil), m.cfg.Infrastructure.Stacks...)
	sort.Slice(stacks, func(i, j int) bool { return stacks[i].StartupOrder < stacks[j].StartupOrder })
	for _, s := range stacks {
		composePath, err := s.ResolvedComposePath(m.home)
		if err != nil {
			return err
		}
		if err := m.dockerCompose(composePath, s.ProjectName, "up", "-d", "--remove-orphans"); err != nil {
			return fmt.Errorf("stack %s: %w", s.Name, err)
		}
		if err := m.waitHealthy(composePath, s.ProjectName, s.ExpectedServices, time.Duration(m.cfg.Infrastructure.StartupTimeoutSeconds)*time.Second); err != nil {
			return fmt.Errorf("stack %s wait: %w", s.Name, err)
		}
	}
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()
	return nil
}

func (m *Manager) StopAll() {
	stacks := append([]config.InfraStack(nil), m.cfg.Infrastructure.Stacks...)
	sort.Slice(stacks, func(i, j int) bool { return stacks[i].StartupOrder < stacks[j].StartupOrder })
	for i := len(stacks) - 1; i >= 0; i-- {
		s := stacks[i]
		composePath, err := s.ResolvedComposePath(m.home)
		if err != nil {
			continue
		}
		_ = m.dockerCompose(composePath, s.ProjectName, "down", "--remove-orphans")
	}
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
}

func (m *Manager) dockerCompose(composeFile, project string, args ...string) error {
	cmdArgs := append([]string{"compose", "-f", composeFile, "-p", project}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, out.String())
	}
	return nil
}

func (m *Manager) waitHealthy(composeFile, project string, expected int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		n, err := m.countRunning(composeFile, project)
		if err == nil && n >= expected {
			return nil
		}
		time.Sleep(time.Duration(m.cfg.Infrastructure.HealthCheckIntervalSeconds) * time.Second)
	}
	return fmt.Errorf("timeout waiting for %d running services", expected)
}

func (m *Manager) countRunning(composeFile, project string) (int, error) {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "-p", project, "ps", "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	n := countRunningJSON(out)
	return n, nil
}

func countRunningJSON(out []byte) int {
	var asArray []map[string]any
	if json.Unmarshal(out, &asArray) == nil {
		return countRunningRows(asArray)
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	var rows []map[string]any
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row map[string]any
		if json.Unmarshal([]byte(line), &row) == nil {
			rows = append(rows, row)
		}
	}
	return countRunningRows(rows)
}

func countRunningRows(rows []map[string]any) int {
	n := 0
	for _, row := range rows {
		st, _ := row["State"].(string)
		if strings.EqualFold(st, "running") {
			n++
			continue
		}
		if s, ok := row["Status"].(string); ok && strings.Contains(strings.ToLower(s), "up") {
			n++
		}
	}
	return n
}

type StackStatus struct {
	Name             string `json:"name"`
	TotalServices    int    `json:"totalServices"`
	RunningServices  int    `json:"runningServices"`
	ExpectedServices int    `json:"expectedServices"`
	HealthStatus     string `json:"healthStatus"`
}

// Status mirrors Java InfrastructureStatus JSON shape.
type Status struct {
	Managed       bool          `json:"managed"`
	OverallHealth string        `json:"overallHealth"`
	Stacks        []StackStatus `json:"stacks"`
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	running := m.running
	m.mu.RUnlock()
	var stacks []StackStatus
	allOK := true
	for _, s := range m.cfg.Infrastructure.Stacks {
		composePath, err := s.ResolvedComposePath(m.home)
		if err != nil {
			stacks = append(stacks, StackStatus{Name: s.Name, ExpectedServices: s.ExpectedServices, HealthStatus: "ERROR: " + err.Error()})
			allOK = false
			continue
		}
		n, err := m.countRunning(composePath, s.ProjectName)
		if err != nil {
			stacks = append(stacks, StackStatus{Name: s.Name, ExpectedServices: s.ExpectedServices, HealthStatus: "ERROR: " + err.Error()})
			allOK = false
			continue
		}
		h := "DEGRADED"
		if n >= s.ExpectedServices {
			h = "HEALTHY"
		} else {
			allOK = false
		}
		stacks = append(stacks, StackStatus{
			Name: s.Name, TotalServices: n, RunningServices: n, ExpectedServices: s.ExpectedServices, HealthStatus: h,
		})
	}
	oh := "HEALTHY"
	if !allOK {
		oh = "DEGRADED"
	}
	return Status{Managed: running, OverallHealth: oh, Stacks: stacks}
}
