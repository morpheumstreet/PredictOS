import Sidebar from "@/components/Sidebar";
import PolymarketArbEngineDashboard from "@/components/PolymarketArbEngineDashboard";

export function DashboardPage() {
  return (
    <div className="flex h-screen overflow-hidden">
      <div className="relative z-10 overflow-visible shrink-0">
        <Sidebar activeTab="dashboard" />
      </div>
      <main className="flex-1 min-w-0 min-h-0 overflow-hidden bg-black">
        <PolymarketArbEngineDashboard />
      </main>
    </div>
  );
}

export default DashboardPage;
