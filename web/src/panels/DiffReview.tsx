import { useState, use, Suspense, useMemo } from 'react';
import { getDatabase, type TaskQueueDocType } from '../db/index';
import { useRxQuery } from '../hooks/useRxQuery';

function DiffReviewContent({ }: { panelId: string }) {
  const db = use(getDatabase());
  const query = useMemo(() => db.task_queue.find({ selector: { column: 'completed' } }), [db]);
  const tasks = useRxQuery<TaskQueueDocType[]>(query as any);
  
  const latestTask = tasks?.find(t => t.extra?.proposedChanges) || null;
  const rawDiff = latestTask?.extra?.proposedChanges || '';
  
  const lines = rawDiff ? rawDiff.split('\n').filter((l: string) => l.length > 0 || l === '') : [];
  const [viewMode, setViewMode] = useState<'unified' | 'split'>('split');

  const handleContextMenu = (e: React.MouseEvent, lineContent: string) => {
    e.preventDefault();
    e.stopPropagation();
    window.dispatchEvent(new CustomEvent('open-context-menu', {
      detail: {
        x: e.clientX,
        y: e.clientY,
        items: [
          { label: 'Copy Line', action: () => navigator.clipboard.writeText(lineContent) },
          { label: 'Copy File Path', action: () => navigator.clipboard.writeText('internal/store/user.go') },
          { divider: true },
          { label: 'View File History' }
        ]
      }
    }));
  };

  const renderSplit = () => {
    const leftLines: { num: number | string, content: string, type: 'del' | 'context' | 'header' }[] = [];
    const rightLines: { num: number | string, content: string, type: 'add' | 'context' | 'header' }[] = [];
    
    let leftNum = 15;
    let rightNum = 15;

    lines.forEach(line => {
      if (line.startsWith('@@')) {
        leftLines.push({ num: '...', content: line, type: 'header' });
        rightLines.push({ num: '...', content: line, type: 'header' });
      } else if (line.startsWith('-')) {
        leftLines.push({ num: leftNum++, content: line, type: 'del' });
      } else if (line.startsWith('+')) {
        rightLines.push({ num: rightNum++, content: line, type: 'add' });
      } else {
        leftLines.push({ num: leftNum++, content: line, type: 'context' });
        rightLines.push({ num: rightNum++, content: line, type: 'context' });
      }
    });

    return (
      <div className="flex w-full h-full font-mono text-[13px] leading-5 text-[#d4d4d4] selection:bg-[#264f78]">
        {/* Left Side (Deletions) */}
        <div className="flex-1 w-1/2 overflow-x-auto border-r border-[#404040] bg-[#1e1e1e]">
          <table className="w-full border-collapse">
            <tbody>
              {leftLines.map((l, i) => (
                <tr key={`l-${i}`} 
                    className={`group ${l.type === 'del' ? 'bg-[#3f2b2b]' : l.type === 'header' ? 'text-[#569cd6] bg-[#2d2d2d]' : 'hover:bg-[#2a2d2e]'}`}
                    onContextMenu={e => handleContextMenu(e, l.content)}>
                  <td className="w-12 text-right pr-3 select-none text-[#6e7681] border-r border-[#404040] bg-[#1e1e1e] sticky left-0">{l.num}</td>
                  <td className="pl-4 whitespace-pre py-px">
                    <span className={l.type === 'del' ? 'bg-[#5c3030] inline-block w-full min-h-[20px]' : ''}>{l.content}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {/* Right Side (Additions) */}
        <div className="flex-1 w-1/2 overflow-x-auto bg-[#1e1e1e]">
          <table className="w-full border-collapse">
            <tbody>
              {rightLines.map((l, i) => (
                <tr key={`r-${i}`} 
                    className={`group ${l.type === 'add' ? 'bg-[#2b3e34]' : l.type === 'header' ? 'text-[#569cd6] bg-[#2d2d2d]' : 'hover:bg-[#2a2d2e]'}`}
                    onContextMenu={e => handleContextMenu(e, l.content)}>
                  <td className="w-12 text-right pr-3 select-none text-[#6e7681] border-r border-[#404040] bg-[#1e1e1e] sticky left-0">{l.num}</td>
                  <td className="pl-4 whitespace-pre py-px">
                    <span className={l.type === 'add' ? 'bg-[#28583c] inline-block w-full min-h-[20px]' : ''}>{l.content}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  };

  const renderUnified = () => {
    let leftNum = 15;
    let rightNum = 15;

    return (
      <div className="w-full h-full overflow-auto bg-[#1e1e1e] font-mono text-[13px] leading-5 text-[#d4d4d4] selection:bg-[#264f78]">
        <table className="w-full border-collapse">
          <tbody>
            {lines.map((line, i) => {
              let lNum: string | number = leftNum;
              let rNum: string | number = rightNum;
              let bgClass = "hover:bg-[#2a2d2e]";
              let lineBgClass = "";

              if (line.startsWith('@@')) {
                bgClass = "text-[#569cd6] bg-[#2d2d2d]";
                lNum = '...'; rNum = '...';
              } else if (line.startsWith('-')) {
                bgClass = "bg-[#3f2b2b]";
                lineBgClass = "bg-[#5c3030] inline-block min-w-full";
                lNum = leftNum++; rNum = '';
              } else if (line.startsWith('+')) {
                bgClass = "bg-[#2b3e34]";
                lineBgClass = "bg-[#28583c] inline-block min-w-full";
                lNum = ''; rNum = rightNum++;
              } else {
                leftNum++; rightNum++;
              }

              return (
                <tr key={i} className={bgClass} onContextMenu={e => handleContextMenu(e, line)}>
                  <td className="w-10 text-right pr-3 select-none text-[#6e7681] border-r border-[#404040] bg-[#1e1e1e] sticky left-0">{lNum}</td>
                  <td className="w-10 text-right pr-3 select-none text-[#6e7681] border-r border-[#404040] bg-[#1e1e1e] sticky left-10">{rNum}</td>
                  <td className="pl-4 whitespace-pre py-px">
                    <span className={lineBgClass}>{line}</span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  };

  return (
    <div className="w-full h-full flex flex-col bg-[#1e1e1e] text-sm"
         onContextMenu={(e) => {
           e.preventDefault();
           window.dispatchEvent(new CustomEvent('open-context-menu', {
             detail: {
               x: e.clientX,
               y: e.clientY,
               items: [
                 { label: viewMode === 'split' ? 'Switch to Unified View' : 'Switch to Split View', action: () => setViewMode(viewMode === 'split' ? 'unified' : 'split') },
                 { label: 'Copy Entire Diff', action: () => navigator.clipboard.writeText(rawDiff) }
               ]
             }
           }));
         }}>
      
      {/* File Header */}
      <div className="h-11 bg-[#252526] border-b border-[#3c3c3c] flex items-center justify-between px-4 shrink-0 shadow-sm z-10">
        <div className="flex items-center gap-3">
          <svg className="w-4 h-4 text-[#519aba]" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
          <span className="font-semibold text-[#cccccc] tracking-wide">internal/store/user.go</span>
          <div className="flex gap-1.5 ml-2 items-center">
            <span className="text-[#81b88b] bg-[#81b88b]/10 px-1.5 py-0.5 rounded text-[11px] font-bold">+2</span>
            <span className="text-[#c74e39] bg-[#c74e39]/10 px-1.5 py-0.5 rounded text-[11px] font-bold">-2</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* View Toggle */}
          <div className="flex bg-[#333333] rounded overflow-hidden border border-[#454545] p-0.5">
            <button 
              onClick={() => setViewMode('split')}
              className={`px-2 py-1 text-[11px] font-medium rounded-sm transition-colors ${viewMode === 'split' ? 'bg-[#094771] text-white' : 'text-[#a0a0a0] hover:text-[#d4d4d4]'}`}
            >
              Split
            </button>
            <button 
              onClick={() => setViewMode('unified')}
              className={`px-2 py-1 text-[11px] font-medium rounded-sm transition-colors ${viewMode === 'unified' ? 'bg-[#094771] text-white' : 'text-[#a0a0a0] hover:text-[#d4d4d4]'}`}
            >
              Unified
            </button>
          </div>
        </div>
      </div>
      
      {/* Diff Content Area */}
      <div className="flex-1 overflow-hidden relative">
        {viewMode === 'split' ? renderSplit() : renderUnified()}
      </div>

      {/* Action Footer */}
      <div className="p-3 bg-[#252526] border-t border-[#3c3c3c] flex justify-end gap-3 shrink-0 items-center">
        <span className="text-xs text-[#858585] mr-auto flex items-center gap-2">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          Reviewing changes from active task: {latestTask?.title || 'None'}
        </span>
        <button className="px-5 py-1.5 bg-transparent text-[#cccccc] border border-[#555555] hover:bg-[#333333] hover:border-[#888888] rounded-md font-medium transition-all text-sm">
          Reject
        </button>
        <button className="px-5 py-1.5 bg-[#0e639c] text-white rounded-md font-medium hover:bg-[#1177bb] transition-all text-sm shadow-lg shadow-[#0e639c]/20 border border-[#1177bb]">
          Approve Change
        </button>
      </div>
    </div>
  );
}

export default function DiffReview({ panelId }: { panelId: string }) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted">Loading diffs...</div>}>
      <DiffReviewContent panelId={panelId} />
    </Suspense>
  );
}
