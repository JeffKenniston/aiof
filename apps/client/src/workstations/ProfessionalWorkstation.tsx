
import AgentDelegationBoard from '../components/professional/AgentDelegationBoard';
import ExecutiveSummaryFeed from '../components/professional/ExecutiveSummaryFeed';
import KPIWidget from '../components/professional/KPIWidget';

export default function ProfessionalWorkstation() {
  return (
    <div className="w-full h-full flex flex-col bg-[#050403] text-text-primary overflow-hidden relative font-sans">
      {/* Background gradients for professional theme (Amber / Gold) */}
      <div className="absolute top-0 right-0 w-[600px] h-[400px] bg-amber-500/5 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute bottom-0 left-0 w-[600px] h-[400px] bg-orange-500/5 rounded-full blur-[120px] pointer-events-none" />

      {/* Header */}
      <header className="h-16 border-b border-white/5 flex items-center justify-between px-8 bg-white/[0.01] z-10 shrink-0">
        <h2 className="text-lg font-medium tracking-wide flex items-center gap-2">
          <span className="text-amber-500">💼</span> Executive Dashboard
        </h2>
        <div className="flex items-center gap-4">
          <div className="text-xs text-text-muted flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            3 Agents Active
          </div>
          <button className="px-4 py-1.5 rounded-full bg-amber-500/10 text-amber-500 border border-amber-500/20 text-xs font-medium hover:bg-amber-500/20 transition-colors shadow-[0_0_15px_rgba(245,158,11,0.15)]">
            New Directive
          </button>
        </div>
      </header>

      {/* Main Dashboard Grid */}
      <div className="flex-1 overflow-y-auto custom-scrollbar p-6 lg:p-8 z-10">
        <div className="max-w-[1600px] mx-auto grid grid-cols-1 xl:grid-cols-12 gap-6 h-full">
          
          {/* Left Column: KPIs and Summary Feed */}
          <div className="xl:col-span-4 flex flex-col gap-6">
            <KPIWidget label="Active Agents" collectionName="agent_logs" icon="clock" />
            <KPIWidget label="Total Actions" collectionName="task_queue" icon="clock" />
            <KPIWidget label="Tasks Completed" collectionName="task_queue" icon="clock" />
            <KPIWidget label="System Messages" collectionName="agent_logs" icon="mail" />
            <div className="flex-1 min-h-0 bg-surface-base/40 backdrop-blur-3xl border border-white/5 rounded-[2rem] overflow-hidden flex flex-col shadow-2xl relative group">
              <div className="absolute inset-0 bg-gradient-to-br from-white/[0.02] to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none" />
              <div className="p-4 border-b border-white/5 bg-white/[0.02]">
                <h3 className="text-xs font-semibold text-text-muted uppercase tracking-widest flex items-center gap-2">
                  <svg className="w-4 h-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                  Executive Briefings
                </h3>
              </div>
              <ExecutiveSummaryFeed />
            </div>
          </div>

          {/* Right Column: Agent Delegation Board */}
          <div className="xl:col-span-8 bg-surface-base/40 backdrop-blur-3xl border border-white/5 rounded-[2rem] overflow-hidden flex flex-col shadow-2xl relative group">
             <div className="absolute inset-0 bg-gradient-to-br from-white/[0.02] to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none" />
            <div className="p-4 border-b border-white/5 bg-white/[0.02] flex items-center justify-between">
              <h3 className="text-xs font-semibold text-text-muted uppercase tracking-widest flex items-center gap-2">
                <svg className="w-4 h-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" /></svg>
                Agent Delegation
              </h3>
              <div className="flex gap-2 relative z-10">
                <button className="p-1.5 hover:bg-white/5 rounded text-text-muted transition-colors"><svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" /></svg></button>
              </div>
            </div>
            <AgentDelegationBoard />
          </div>

        </div>
      </div>
    </div>
  );
}
