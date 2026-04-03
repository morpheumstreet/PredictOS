import { Navigate, Route, Routes } from "react-router-dom";
import AgenticMarketAnalysis from "@/components/AgenticMarketAnalysis";
import ArbitrageTerminal from "@/components/ArbitrageTerminal";
import EventScannerTerminal from "@/components/EventScannerTerminal";
import Sidebar from "@/components/Sidebar";
import WalletTrackingTerminal from "@/components/WalletTrackingTerminal";
import { BettingBotsPage } from "@/pages/BettingBotsPage";
import { AgentsPage } from "@/pages/AgentsPage";

function AnalysisPage() {
  return (
    <div className="flex h-screen">
      <div className="relative z-10 overflow-visible">
        <Sidebar activeTab="analysis" />
      </div>
      <main className="flex-1 overflow-y-auto overflow-x-hidden">
        <AgenticMarketAnalysis />
      </main>
    </div>
  );
}

function ArbitragePage() {
  return (
    <div className="flex h-screen">
      <div className="relative z-10 overflow-visible">
        <Sidebar activeTab="arbitrage" />
      </div>
      <main className="flex-1 overflow-y-auto overflow-x-hidden">
        <ArbitrageTerminal />
      </main>
    </div>
  );
}

function WalletTrackingPage() {
  return (
    <div className="flex h-screen">
      <div className="relative z-10 overflow-visible">
        <Sidebar activeTab="wallet-tracking" />
      </div>
      <main className="flex-1 overflow-y-auto overflow-x-hidden">
        <WalletTrackingTerminal />
      </main>
    </div>
  );
}

function EventScannerPage() {
  return (
    <div className="flex h-screen">
      <div className="relative z-10 overflow-visible">
        <Sidebar activeTab="event-scanner" />
      </div>
      <main className="flex-1 overflow-y-auto overflow-x-hidden">
        <EventScannerTerminal />
      </main>
    </div>
  );
}

export function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/market-analysis" replace />} />
      <Route path="/market-analysis" element={<AnalysisPage />} />
      <Route path="/arbitrage" element={<ArbitragePage />} />
      <Route path="/betting-bots" element={<BettingBotsPage />} />
      <Route path="/wallet-tracking" element={<WalletTrackingPage />} />
      <Route path="/event-scanner" element={<EventScannerPage />} />
      <Route path="/agents" element={<AgentsPage />} />
      <Route path="*" element={<Navigate to="/market-analysis" replace />} />
    </Routes>
  );
}
