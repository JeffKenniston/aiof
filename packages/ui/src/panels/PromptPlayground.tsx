import { useState } from 'react';
import { pushHandler } from "@aiof/rxdb-store";
import { Document } from '../gen/proto/orchestrator/v1/orchestrator_pb';

export default function PromptPlayground({  }: { panelId: string }) {
  const [prompt, setPrompt] = useState('Write a hello world in Go.');
  const [model, setModel] = useState('gemini-3.5-flash');
  const [compareMode, setCompareMode] = useState(false);
  const [response, setResponse] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [latency, setLatency] = useState<number | null>(null);

  const handleSend = async () => {
    if (!prompt) return;
    setIsGenerating(true);
    setResponse('');
    setLatency(null);
    const start = Date.now();
    try {
      const docPayload = JSON.stringify({
          task_id: "playground-task-" + Date.now(),
          column: "active",
          title: prompt,
          project_id: "default",
          created_at: Date.now()
      });
      
      const doc = new Document({
          id: "task-" + Date.now(),
          documentType: "task_queue",
          payload: new TextEncoder().encode(docPayload),
          updatedAt: BigInt(Date.now()),
          isDeleted: false
      });
      
      const success = await pushHandler([doc]);
      if (success) {
        setResponse('Task submitted! Watch the telemetry graph for real-time routing status.');
      } else {
        setResponse('Failed to submit task.');
      }
    } catch (err: any) {
      setResponse('Error: ' + err.message);
    } finally {
      setLatency(Date.now() - start);
      setIsGenerating(false);
    }
  };

  return (
    <div className="w-full h-full flex flex-col md:flex-row bg-surface-base text-sm">
      {/* Left Pane: Input */}
      <div className="w-full md:w-1/2 flex flex-col border-r border-border-default">
        <div className="p-4 border-b border-border-default bg-surface-raised flex items-center justify-between shrink-0">
          <select 
            className="bg-surface-base border border-border-default text-text-primary rounded px-2 py-1 focus:outline-none focus:border-accent-blue"
            value={model}
            onChange={e => setModel(e.target.value)}
          >
            <option value="gemini-3.1-pro-preview">gemini-3.1-pro-preview</option>
            <option value="gemini-3.5-flash">gemini-3.5-flash</option>
          </select>
          <div className="flex items-center gap-2">
            <label className="flex items-center gap-2 cursor-pointer text-text-secondary text-xs">
              <input 
                type="checkbox" 
                className="rounded border-border-default text-accent-blue focus:ring-accent-blue bg-surface-base"
                checked={compareMode}
                onChange={e => setCompareMode(e.target.checked)}
              />
              Compare Models
            </label>
            <button 
              onClick={handleSend}
              disabled={isGenerating || !prompt}
              className="px-4 py-1.5 bg-accent-blue text-white rounded font-medium hover:bg-blue-600 disabled:opacity-50 transition-colors shadow-lg shadow-blue-500/20 flex items-center gap-2"
            >
              {isGenerating ? 'Sending...' : 'Send'}
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" /></svg>
            </button>
          </div>
        </div>
        <div className="flex-1 p-4 bg-surface-raised">
          <textarea
            className="w-full h-full bg-surface-base border border-border-default rounded p-3 text-text-primary resize-none focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue transition-all"
            placeholder="Enter prompt..."
            value={prompt}
            onChange={e => setPrompt(e.target.value)}
          />
        </div>
      </div>

      {/* Right Pane: Output */}
      <div className="w-full md:w-1/2 flex flex-col bg-surface-base">
        <div className="p-2 border-b border-border-default bg-surface-raised flex justify-end gap-4 text-xs text-text-muted shrink-0">
          <span>Tokens: {response ? Math.floor(response.length / 4) : 0}</span>
          <span>Latency: {latency ? `${latency}ms` : '--'}</span>
        </div>
        <div className="flex-1 p-4 overflow-y-auto font-mono text-xs whitespace-pre-wrap text-text-primary">
          {isGenerating ? (
            <div className="flex items-center gap-3 text-accent-blue h-full justify-center">
              <div className="w-5 h-5 border-2 border-accent-blue border-t-transparent rounded-full animate-spin" />
              Generating...
            </div>
          ) : response ? (
            response
          ) : (
            <div className="text-text-muted flex h-full items-center justify-center italic">Response will appear here...</div>
          )}
        </div>
      </div>
    </div>
  );
}
