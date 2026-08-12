import { useState, useEffect } from 'react';
import { BACKEND_URL } from '../config';
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from '@xyflow/react';

export default function KnowledgeGraph({  }: { panelId: string }) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    fetch(`${BACKEND_URL}/api/graph`)
      .then(res => res.json())
      .then(data => {
        if (data.nodes && data.edges) {
          setNodes(data.nodes);
          setEdges(data.edges);
        }
      })
      .catch(err => {
        console.warn('Failed to fetch graph data:', err);
        setNodes([]);
        setEdges([]);
      });
  }, [setNodes, setEdges]);

  return (
    <div className="w-full h-full bg-surface-base flex flex-col relative">
      <div className="absolute top-4 left-4 z-10">
        <input 
          type="text"
          placeholder="Search entities..."
          className="bg-surface-raised border border-border-default text-text-primary px-3 py-1.5 rounded-lg shadow-elevated focus:outline-none focus:border-accent-blue"
          value={searchTerm}
          onChange={e => setSearchTerm(e.target.value)}
        />
      </div>
      <ReactFlow
        nodes={nodes.map(n => ({
          ...n,
          style: {
            ...n.style,
            opacity: searchTerm && !(n.data.label as string).toLowerCase().includes(searchTerm.toLowerCase()) ? 0.2 : 1
          }
        }))}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#ffffff" gap={20} size={1} style={{ opacity: 0.05 }} />
        <Controls className="fill-text-primary bg-surface-raised border border-border-default shadow-elevated rounded-md" />
      </ReactFlow>
    </div>
  );
}
