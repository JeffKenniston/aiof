import { useCallback } from 'react';
import {
  ReactFlow,
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Handle,
  Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

// Custom Agent Node Component
function AgentNode({ data }: { data: any }) {
  return (
    <div className="px-4 py-2 shadow-panel rounded-panel bg-surface-raised border border-border-default min-w-[150px] relative">
      <Handle type="target" position={Position.Top} className="w-2 h-2 bg-accent-blue" />
      <div className="flex flex-col items-center">
        <div className="text-xs font-bold text-text-secondary uppercase mb-1">{data.role as string}</div>
        <div className="text-sm font-medium text-text-primary">{data.label as string}</div>
      </div>
      <Handle type="source" position={Position.Bottom} className="w-2 h-2 bg-accent-crimson" />
      {data.status === 'active' && (
        <div className="absolute -top-1 -right-1 w-3 h-3 rounded-full bg-accent-crimson animate-pulse border-2 border-surface-raised" />
      )}
    </div>
  );
}

const nodeTypes = {
  agent: AgentNode,
};

const initialNodes = [
  { id: '1', type: 'agent', position: { x: 250, y: 50 }, data: { label: 'Router', role: 'Orchestrator', status: 'active' } },
  { id: '2', type: 'agent', position: { x: 100, y: 150 }, data: { label: 'Architect', role: 'System Design', status: 'idle' } },
  { id: '3', type: 'agent', position: { x: 400, y: 150 }, data: { label: 'SandboxOrchestrator', role: 'Execution', status: 'idle' } },
  { id: '4', type: 'agent', position: { x: 100, y: 250 }, data: { label: 'DevOps', role: 'Deployment', status: 'idle' } },
  { id: '5', type: 'agent', position: { x: 400, y: 250 }, data: { label: 'Dynamic Agent', role: 'Fabricator', status: 'active' } },
];

const initialEdges = [
  { id: 'e1-2', source: '1', target: '2', animated: false, style: { stroke: 'rgba(255,255,255,0.1)' } },
  { id: 'e1-3', source: '1', target: '3', animated: true, style: { stroke: '#DC143C', strokeWidth: 2 } },
  { id: 'e2-4', source: '2', target: '4', animated: false, style: { stroke: 'rgba(255,255,255,0.1)' } },
  { id: 'e3-5', source: '3', target: '5', animated: true, style: { stroke: '#DC143C', strokeWidth: 2 } },
];

export default function SpatialCanvas() {
  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const onConnect = useCallback(
    (params: any) => setEdges((eds) => addEdge({ ...params, style: { stroke: '#DC143C' }, animated: true }, eds)),
    [setEdges],
  );

  return (
    <div className="w-full h-full bg-surface-base relative border-r border-border-default">
      <div className="absolute top-4 left-4 z-10 px-3 py-1 bg-surface-raised border border-border-default rounded-pill text-xs font-medium text-text-secondary flex items-center gap-2 shadow-panel">
        <div className="w-2 h-2 rounded-full bg-accent-crimson animate-pulse" />
        Spatial Canvas Active
      </div>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        fitView
        colorMode="dark"
        minZoom={0.5}
      >
        <Controls showInteractive={false} />
        <MiniMap 
          nodeColor={(n) => {
            if (n.data?.status === 'active') return '#DC143C';
            return '#1c1c1c';
          }}
          maskColor="rgba(18, 18, 18, 0.7)"
          style={{ backgroundColor: '#121212', border: '1px solid rgba(255,255,255,0.05)' }}
        />
        <Background color="#262626" gap={16} />
      </ReactFlow>
    </div>
  );
}
