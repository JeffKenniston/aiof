import { lazy } from 'react';
import type { PanelDefinition } from './types';

export const PANEL_REGISTRY: Record<string, PanelDefinition> = {
  'agent-visual': {
    id: 'agent-visual',
    title: 'Agent Orchestration',
    icon: '🔮',
    category: 'core',
    component: lazy(() => import('../panels/AgentVisual')),
  },
  'agent-text': {
    id: 'agent-text',
    title: 'Agent Log',
    icon: '📜',
    category: 'core',
    component: lazy(() => import('../panels/AgentTextLog')),
  },
  'terminal': {
    id: 'terminal',
    title: 'Terminal',
    icon: '⬛',
    category: 'core',
    component: lazy(() => import('../panels/Terminal')),
  },
  'code-editor': {
    id: 'code-editor',
    title: 'Code Editor',
    icon: '✏️',
    category: 'core',
    component: lazy(() => import('../panels/CodeEditor')),
  },
  'settings': {
    id: 'settings',
    title: 'Settings',
    icon: '⚙️',
    category: 'core',
    component: lazy(() => import('../panels/Settings')),
  },
  'wal-inspector': {
    id: 'wal-inspector',
    title: 'WAL Inspector',
    icon: '🔍',
    category: 'power',
    component: lazy(() => import('../panels/WALInspector')),
  },
  'sandbox-monitor': {
    id: 'sandbox-monitor',
    title: 'Sandbox Monitor',
    icon: '📦',
    category: 'power',
    component: lazy(() => import('../panels/SandboxMonitor')),
  },
  'telemetry': {
    id: 'telemetry',
    title: 'Telemetry',
    icon: '📊',
    category: 'power',
    component: lazy(() => import('../panels/Telemetry')),
  },
  'knowledge-graph': {
    id: 'knowledge-graph',
    title: 'Knowledge Graph',
    icon: '🧠',
    category: 'power',
    component: lazy(() => import('../panels/KnowledgeGraph')),
  },
  'context-cache': {
    id: 'context-cache',
    title: 'Context Cache',
    icon: '💾',
    category: 'power',
    component: lazy(() => import('../panels/ContextCache')),
  },
  'prompt-playground': {
    id: 'prompt-playground',
    title: 'Prompt Playground',
    icon: '🧪',
    category: 'specialized',
    component: lazy(() => import('../panels/PromptPlayground')),
  },
  'diff-review': {
    id: 'diff-review',
    title: 'Diff / Review',
    icon: '🔀',
    category: 'specialized',
    component: lazy(() => import('../panels/DiffReview')),
  },
  'chat': {
    id: 'chat',
    title: 'Chat',
    icon: '💬',
    category: 'specialized',
    component: lazy(() => import('../panels/Chat')),
  },
  'task-queue': {
    id: 'task-queue',
    title: 'Task Queue',
    icon: '📋',
    category: 'specialized',
    component: lazy(() => import('../panels/TaskQueue')),
  },
  'file-explorer': {
    id: 'file-explorer',
    title: 'File Explorer',
    icon: '📁',
    category: 'specialized',
    component: lazy(() => import('../panels/FileExplorer')),
  },
  'schema-inspector': {
    id: 'schema-inspector',
    title: 'Schema Inspector',
    icon: '🏗️',
    category: 'specialized',
    component: lazy(() => import('../panels/SchemaInspector')),
  },
};

export function getPanelsByCategory(category: PanelDefinition['category']): PanelDefinition[] {
  return Object.values(PANEL_REGISTRY).filter(p => p.category === category);
}
