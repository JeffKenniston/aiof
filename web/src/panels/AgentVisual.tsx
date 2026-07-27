import { useEffect, memo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  Handle,
  Position,
  type NodeProps,
} from '@xyflow/react';
import { getDatabase, type AgentGraphStateDocType } from '../db/index';
import { useRxQuery } from '../hooks/useRxQuery';
import { use, Suspense, useMemo } from 'react';

interface AgentVisualProps {
  panelId: string;
}

interface AgentNodeData extends Record<string, unknown> {
  label: string;
  model: string;
  color: string;
  status: 'idle' | 'running' | 'complete';
}

const STATUS_COLORS: Record<'idle' | 'running' | 'complete', string> = {
  idle: 'bg-surface-overlay border-border-default',
  running: 'bg-accent-blue/20 border-accent-blue shadow-[0_0_15px_rgba(59,130,246,0.5)]',
  complete: 'bg-accent-green/20 border-accent-green',
};

const CustomNode = memo(({ data }: NodeProps<Node<AgentNodeData>>) => {
  const statusClass = STATUS_COLORS[data.status] || STATUS_COLORS.idle;
  
  return (
    <div className={`px-4 py-3 rounded-xl border backdrop-blur-md transition-all duration-300 ${statusClass} min-w-[180px]`}>
      <Handle type="target" position={Position.Top} className="w-2 h-2 bg-text-muted border-none" />
      
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full" style={{ backgroundColor: data.color }} />
          <span className="font-semibold text-text-primary text-sm">{data.label}</span>
        </div>
        <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-surface-base text-text-muted border border-border-default">
          {data.model}
        </span>
      </div>
      
      <div className="flex justify-between items-center text-xs">
        <span className="text-text-muted">Status:</span>
        <span className={`capitalize font-medium ${data.status === 'running' ? 'text-accent-blue animate-pulse' : 'text-text-secondary'}`}>
          {data.status}
        </span>
      </div>

      <Handle type="source" position={Position.Bottom} className="w-2 h-2 bg-text-muted border-none" />
    </div>
  );
});

const nodeTypes = { custom: CustomNode };

const initialNodes: Node<AgentNodeData>[] = [
  { id: 'router', type: 'custom', position: { x: 250, y: 50 }, data: { label: 'Router', model: 'Pro', color: '#58a6ff', status: 'idle' } },
  { id: 'sys-arch', type: 'custom', position: { x: 50, y: 200 }, data: { label: 'Sys Architecture', model: 'Pro', color: '#3fb950', status: 'idle' } },
  { id: 'data-arch', type: 'custom', position: { x: 250, y: 200 }, data: { label: 'Data Architecture', model: 'Pro', color: '#3fb950', status: 'idle' } },
  { id: 'infra', type: 'custom', position: { x: 450, y: 200 }, data: { label: 'Infrastructure', model: 'Pro', color: '#3fb950', status: 'idle' } },
  { id: 'validator', type: 'custom', position: { x: 150, y: 350 }, data: { label: 'Validator', model: 'Flash', color: '#d29922', status: 'idle' } },
  { id: 'executor', type: 'custom', position: { x: 350, y: 350 }, data: { label: 'Executor', model: 'Flash', color: '#f85149', status: 'idle' } },
];

const initialEdges: Edge[] = [
  { id: 'e1', source: 'router', target: 'sys-arch', animated: true, style: { stroke: '#58a6ff' } },
  { id: 'e2', source: 'router', target: 'data-arch', animated: true, style: { stroke: '#58a6ff' } },
  { id: 'e3', source: 'router', target: 'infra', animated: true, style: { stroke: '#58a6ff' } },
  { id: 'e4', source: 'sys-arch', target: 'validator', animated: true, style: { stroke: '#3fb950' } },
  { id: 'e5', source: 'data-arch', target: 'validator', animated: true, style: { stroke: '#3fb950' } },
  { id: 'e6', source: 'infra', target: 'validator', animated: true, style: { stroke: '#3fb950' } },
  { id: 'e7', source: 'validator', target: 'executor', animated: true, style: { stroke: '#d29922' } },
];

function AgentVisualContent({ }: AgentVisualProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node<AgentNodeData>>(initialNodes);
  const [edges, , onEdgesChange] = useEdgesState<Edge>(initialEdges);

  const db = use(getDatabase());
  const query = useMemo(() => db.agent_graph_state.find(), [db]);
  const docs = useRxQuery<AgentGraphStateDocType[]>(query as any);

  useEffect(() => {
    if (docs && docs.length > 0) {
      setNodes(nds => nds.map(node => {
        const doc = docs.find((d: any) => d.id === node.id);
        if (doc) {
          return { ...node, data: { ...node.data, status: doc.status as any } };
        }
        return node;
      }));
    }
  }, [docs, setNodes]);

  return (
    <div className="w-full h-full bg-surface-base">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#ffffff" gap={16} size={1} style={{ opacity: 0.05 }} />
        <Controls className="fill-text-primary bg-surface-raised border border-border-default shadow-elevated rounded-md" />
      </ReactFlow>
    </div>
  );
}

export default function AgentVisual(props: AgentVisualProps) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted">Loading Agent Graph...</div>}>
      <AgentVisualContent {...props} />
    </Suspense>
  );
}
