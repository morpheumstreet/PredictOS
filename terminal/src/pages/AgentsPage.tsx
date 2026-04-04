import Sidebar from "@/components/Sidebar";
import { AgentsErrorBanner } from "./agents/AgentsErrorBanner";
import { AgentsPageHeader } from "./agents/AgentsPageHeader";
import { StrategiesTable } from "./agents/StrategiesTable";
import { StrategyEditorModal } from "./agents/StrategyEditorModal";
import { useAgentsPage } from "./agents/useAgentsPage";

export function AgentsPage() {
  const {
    strategies,
    loading,
    error,
    saving,
    panel,
    form,
    modalTab,
    intentDraft,
    generating,
    strategyStatus,
    strategyStatusError,
    strategyStatusLoading,
    setForm,
    setModalTab,
    setIntentDraft,
    openNew,
    openEdit,
    closePanel,
    runExpand,
    toggleTarget,
    submit,
    remove,
    refreshStrategyStatus,
  } = useAgentsPage();

  return (
    <div className="flex h-screen">
      <div className="relative z-10 overflow-visible">
        <Sidebar activeTab="agents" />
      </div>
      <main className="flex-1 overflow-y-auto overflow-x-hidden p-6 space-y-6">
        <AgentsPageHeader onNew={openNew} />
        <AgentsErrorBanner message={error} />
        <StrategiesTable
          strategies={strategies}
          loading={loading}
          onEdit={openEdit}
          onDelete={remove}
        />
        <StrategyEditorModal
          panel={panel}
          modalTab={modalTab}
          form={form}
          intentDraft={intentDraft}
          generating={generating}
          saving={saving}
          strategyStatus={strategyStatus}
          strategyStatusError={strategyStatusError}
          strategyStatusLoading={strategyStatusLoading}
          onClose={closePanel}
          setModalTab={setModalTab}
          setIntentDraft={setIntentDraft}
          setForm={setForm}
          toggleTarget={toggleTarget}
          onExpand={runExpand}
          onSubmit={submit}
          onRefreshStrategyStatus={refreshStrategyStatus}
        />
      </main>
    </div>
  );
}
