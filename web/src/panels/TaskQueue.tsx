import { use, Suspense, useMemo } from 'react';
import { getDatabase, type TaskQueueDocType } from '../db/index';
import { useRxQuery } from '../hooks/useRxQuery';

function TaskQueueContent({ }: { panelId: string }) {
  const db = use(getDatabase());
  const query = useMemo(() => db.task_queue.find(), [db]);
  const tasks = useRxQuery<TaskQueueDocType[]>(query as any);

  const columns = [
    { id: 'pending', title: 'Pending', borderColor: 'border-border-default', bgHighlight: 'hover:border-border-active' },
    { id: 'active', title: 'Active', borderColor: 'border-accent-blue', bgHighlight: 'shadow-[0_0_15px_rgba(59,130,246,0.15)]' },
    { id: 'completed', title: 'Completed', borderColor: 'border-accent-green', bgHighlight: '' },
  ];

  return (
    <div className="w-full h-full bg-surface-base p-4 flex flex-col">
      <div className="flex justify-between items-center mb-4 shrink-0">
        <h2 className="text-lg font-semibold text-text-primary">Agent Task Queue</h2>
        <button className="px-3 py-1.5 bg-accent-blue text-white rounded text-sm font-medium hover:bg-blue-600 transition-colors shadow-lg shadow-blue-500/20">
          + Add Task
        </button>
      </div>

      <div className="flex-1 min-h-0 flex gap-4 overflow-x-auto pb-2">
        {columns.map(col => {
          const colTasks = tasks.filter(t => t.column === col.id);
          
          return (
            <div key={col.id} className="flex-1 min-w-[280px] flex flex-col bg-surface-overlay border border-border-default rounded-xl overflow-hidden">
              <div className={`p-3 border-b border-border-default font-semibold text-text-primary flex justify-between items-center bg-surface-raised`}>
                <div className="flex items-center gap-2">
                  <div className={`w-2 h-2 rounded-full bg-${col.borderColor.split('-')[1]}-${col.borderColor.split('-')[2] || 'default'}`} />
                  {col.title}
                </div>
                <span className="text-xs bg-surface-base px-2 py-0.5 rounded-full text-text-muted border border-border-subtle">
                  {colTasks.length}
                </span>
              </div>
              
              <div className="flex-1 p-2 space-y-2 overflow-y-auto">
                {colTasks.map(task => (
                  <div 
                    key={task.id} 
                    className={`bg-surface-base border ${col.id === 'active' ? 'border-accent-blue' : 'border-border-default'} p-3 rounded-lg ${col.bgHighlight} transition-all cursor-pointer group`}
                    onContextMenu={(e) => {
                      e.preventDefault();
                      window.dispatchEvent(new CustomEvent('open-context-menu', {
                        detail: {
                          x: e.clientX,
                          y: e.clientY,
                          items: [
                            { label: 'Inspect in Panel', action: () => alert(`Feature coming soon for: ${task.title}`) },
                            { label: 'Cycle Priority', action: async () => {
                                const db = await getDatabase();
                                const doc = await db.task_queue.findOne(task.id).exec();
                                if (doc) {
                                  const priorities = ['low', 'medium', 'high'];
                                  const next = priorities[(priorities.indexOf(doc.priority || 'low') + 1) % priorities.length];
                                  await doc.patch({ priority: next });
                                }
                            }},
                            { divider: true },
                            { label: 'Delete Task', danger: true, action: async () => {
                                if (confirm(`Delete task: ${task.title}?`)) {
                                  const db = await getDatabase();
                                  const doc = await db.task_queue.findOne(task.id).exec();
                                  if (doc) await doc.remove();
                                }
                            }}
                          ]
                        }
                      }));
                    }}
                  >
                    <div className="flex justify-between items-start mb-2">
                      <h4 className="text-sm font-medium text-text-primary group-hover:text-accent-blue transition-colors">{task.title}</h4>
                      <div className="flex gap-1">
                        {task.priority === 'high' && <div className="w-2 h-2 rounded-full bg-accent-red" title="High Priority" />}
                        {task.priority === 'medium' && <div className="w-2 h-2 rounded-full bg-accent-amber" title="Medium Priority" />}
                        {task.priority === 'low' && <div className="w-2 h-2 rounded-full bg-accent-green" title="Low Priority" />}
                      </div>
                    </div>
                    
                    <div className="flex items-center justify-between mt-3 text-xs">
                      <div className="flex items-center gap-1.5">
                        <div className="w-5 h-5 rounded-full bg-surface-raised border border-border-default flex items-center justify-center text-[10px] font-bold text-text-secondary">
                          {task.agent ? task.agent.charAt(0).toUpperCase() : '?'}
                        </div>
                        <span className="text-text-secondary">{task.agent || 'Unassigned'}</span>
                      </div>
                      <span className="text-text-muted font-mono">{task.time}</span>
                    </div>
                  </div>
                ))}
                {colTasks.length === 0 && (
                  <div className="text-center p-4 text-xs text-text-muted border border-dashed border-border-subtle rounded-lg">
                    No tasks
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default function TaskQueue({ panelId }: { panelId: string }) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted">Loading Task Queue...</div>}>
      <TaskQueueContent panelId={panelId} />
    </Suspense>
  );
}
