import { useState, useEffect } from 'react';
import { BACKEND_URL } from '../config';
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  MiniMap,
  type Node,
  type Edge,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { Network, Search, Filter } from 'lucide-react';

export default function KnowledgeGraph() {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    fetch(`${BACKEND_URL}/api/graph`)
      .then(res => res.json())
      .then(data => {
        if (data.nodes && data.edges) {
          // Apply custom cool styling to nodes
          const styledNodes = data.nodes.map((n: Node) => ({
            ...n,
            style: {
              background: 'rgba(20, 20, 25, 0.9)',
              color: '#fff',
              border: '1px solid rgba(168, 85, 247, 0.4)',
              borderRadius: '8px',
              padding: '10px 15px',
              fontSize: '12px',
              boxShadow: '0 0 15px rgba(168, 85, 247, 0.2)',
              backdropFilter: 'blur(10px)',
              width: 150,
            }
          }));
          const styledEdges = data.edges.map((e: Edge) => ({
            ...e,
            style: { stroke: 'rgba(59, 130, 246, 0.5)', strokeWidth: 2 },
            animated: true,
          }));
          setNodes(styledNodes);
          setEdges(styledEdges);
        }
      })
      .catch(err => {
        console.warn('Failed to fetch graph data, using mock data:', err);
        // Fallback mock graph data for GraphRAG
        const mockNodes = [
          { id: '1', position: { x: 250, y: 0 }, data: { label: 'Auth Middleware' }, style: { background: '#1e1e1e', color: '#fff', border: '1px solid #a855f7', borderRadius: '8px', boxShadow: '0 0 15px rgba(168, 85, 247, 0.4)' } },
          { id: '2', position: { x: 100, y: 100 }, data: { label: 'User Controller' }, style: { background: '#1e1e1e', color: '#fff', border: '1px solid #3b82f6', borderRadius: '8px', boxShadow: '0 0 15px rgba(59, 130, 246, 0.4)' } },
          { id: '3', position: { x: 400, y: 100 }, data: { label: 'Role Manager' }, style: { background: '#1e1e1e', color: '#fff', border: '1px solid #3b82f6', borderRadius: '8px', boxShadow: '0 0 15px rgba(59, 130, 246, 0.4)' } },
          { id: '4', position: { x: 250, y: 200 }, data: { label: 'Database Core' }, style: { background: '#1e1e1e', color: '#fff', border: '1px solid #10b981', borderRadius: '8px', boxShadow: '0 0 15px rgba(16, 185, 129, 0.4)' } },
        ];
        const mockEdges = [
          { id: 'e1-2', source: '1', target: '2', animated: true, style: { stroke: 'rgba(168,85,247,0.5)', strokeWidth: 2 } },
          { id: 'e1-3', source: '1', target: '3', animated: true, style: { stroke: 'rgba(168,85,247,0.5)', strokeWidth: 2 } },
          { id: 'e2-4', source: '2', target: '4', animated: true, style: { stroke: 'rgba(59,130,246,0.5)', strokeWidth: 2 } },
          { id: 'e3-4', source: '3', target: '4', animated: true, style: { stroke: 'rgba(59,130,246,0.5)', strokeWidth: 2 } },
        ];
        setNodes(mockNodes);
        setEdges(mockEdges);
      });
  }, [setNodes, setEdges]);

  return (
    <div className="w-full h-full bg-neutral-950 flex flex-col relative overflow-hidden">
      <div className="absolute top-4 left-4 right-4 z-10 flex gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-2.5 w-4 h-4 text-neutral-400" />
          <input 
            type="text"
            placeholder="Search Semantic Graph..."
            className="w-full bg-black/40 border border-white/10 text-white pl-9 pr-3 py-2 text-sm rounded-lg shadow-lg focus:outline-none focus:border-accent-purple/50 backdrop-blur-md"
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
          />
        </div>
        <button className="px-3 py-2 bg-black/40 border border-white/10 rounded-lg text-neutral-400 hover:text-white hover:bg-white/10 transition-colors shadow-lg backdrop-blur-md">
          <Filter className="w-4 h-4" />
        </button>
      </div>

      <div className="absolute bottom-4 left-4 z-10 bg-black/40 border border-white/10 px-3 py-2 rounded-lg backdrop-blur-md shadow-lg flex items-center gap-2">
        <Network className="w-4 h-4 text-accent-purple" />
        <span className="text-xs font-semibold text-neutral-200 uppercase tracking-widest">GraphRAG Explorer</span>
      </div>

      <ReactFlow
        nodes={nodes.map(n => ({
          ...n,
          style: {
            ...n.style,
            opacity: searchTerm && !(n.data.label as string).toLowerCase().includes(searchTerm.toLowerCase()) ? 0.1 : 1,
            transform: searchTerm && (n.data.label as string).toLowerCase().includes(searchTerm.toLowerCase()) ? 'scale(1.05)' : 'scale(1)',
            transition: 'all 0.3s ease'
          }
        }))}
        edges={edges.map(e => ({
          ...e,
          style: {
            ...e.style,
            opacity: searchTerm ? 0.1 : 1,
            transition: 'opacity 0.3s ease'
          }
        }))}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        fitView
        colorMode="dark"
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#ffffff" gap={24} size={1} style={{ opacity: 0.03 }} />
        <Controls className="fill-white bg-black/40 border-none shadow-lg !rounded-lg overflow-hidden backdrop-blur-md [&>button]:border-white/10 [&>button]:bg-transparent hover:[&>button]:bg-white/10 [&>button]:text-white" />
        <MiniMap 
          className="bg-black/40 border border-white/10 rounded-lg !shadow-lg backdrop-blur-md overflow-hidden" 
          maskColor="rgba(0, 0, 0, 0.5)"
          nodeColor={(n) => {
            if (n.style?.border?.toString().includes('a855f7')) return '#a855f7';
            if (n.style?.border?.toString().includes('10b981')) return '#10b981';
            return '#3b82f6';
          }}
        />
      </ReactFlow>
    </div>
  );
}
