import type { WorkspaceLayout } from './types';

export const PRESETS: WorkspaceLayout[] = [
  {
    id: 'casual',
    name: 'Casual',
    icon: '⚡',
    description: 'Standard everyday AI assistant workflow',
    layouts: {
      'semi-full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'prompt-playground', col: 0, row: 0, colSpan: 6, rowSpan: 12 },
          { panelId: 'knowledge-graph', col: 6, row: 0, colSpan: 6, rowSpan: 12 },
        ]
      },
      'full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'prompt-playground', col: 0, row: 0, colSpan: 6, rowSpan: 6 },
          { panelId: 'knowledge-graph', col: 0, row: 6, colSpan: 6, rowSpan: 6 },
          { panelId: 'telemetry', col: 6, row: 0, colSpan: 6, rowSpan: 12 },
        ]
      },
      'expanded': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'prompt-playground', col: 0, row: 0, colSpan: 4, rowSpan: 6 },
          { panelId: 'knowledge-graph', col: 0, row: 6, colSpan: 4, rowSpan: 6 },
          { panelId: 'context-cache', col: 4, row: 0, colSpan: 4, rowSpan: 12 },
          { panelId: 'agent-visual', col: 8, row: 0, colSpan: 4, rowSpan: 4 },
          { panelId: 'agent-text', col: 8, row: 4, colSpan: 4, rowSpan: 4 },
          { panelId: 'telemetry', col: 8, row: 8, colSpan: 4, rowSpan: 4 },
        ]
      }
    },
    availablePanels: ['knowledge-graph', 'prompt-playground', 'telemetry'],
    defaultPanelId: 'prompt-playground'
  },
  {
    id: 'dev',
    name: 'Dev',
    icon: '🛠️',
    description: 'Software development workstation',
    layouts: {
      'semi-full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'file-explorer', col: 0, row: 0, colSpan: 3, rowSpan: 12 },
          { panelId: 'code-editor', col: 3, row: 0, colSpan: 9, rowSpan: 9 },
          { panelId: 'terminal', col: 3, row: 9, colSpan: 9, rowSpan: 3 },
        ]
      },
      'full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'file-explorer', col: 0, row: 0, colSpan: 2, rowSpan: 12 },
          { panelId: 'code-editor', col: 2, row: 0, colSpan: 10, rowSpan: 9 },
          { panelId: 'terminal', col: 2, row: 9, colSpan: 10, rowSpan: 3 },
        ]
      },
      'expanded': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'file-explorer', col: 0, row: 0, colSpan: 3, rowSpan: 12 },
          { panelId: 'code-editor', col: 3, row: 0, colSpan: 9, rowSpan: 9 },
          { panelId: 'terminal', col: 3, row: 9, colSpan: 9, rowSpan: 3 },
        ]
      }
    },
    availablePanels: ['code-editor', 'terminal', 'file-explorer', 'diff-review', 'sandbox-monitor'],
    defaultPanelId: 'code-editor'
  },
  {
    id: 'work',
    name: 'Work',
    icon: '💼',
    description: 'Deep research and professional compilation',
    layouts: {
      'semi-full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'file-explorer', col: 0, row: 0, colSpan: 3, rowSpan: 12 },
          { panelId: 'task-queue', col: 3, row: 0, colSpan: 9, rowSpan: 12 },
        ]
      },
      'full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'file-explorer', col: 0, row: 0, colSpan: 3, rowSpan: 12 },
          { panelId: 'task-queue', col: 3, row: 0, colSpan: 9, rowSpan: 6 },
          { panelId: 'agent-text', col: 3, row: 6, colSpan: 9, rowSpan: 6 },
        ]
      },
      'expanded': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'file-explorer', col: 0, row: 0, colSpan: 2, rowSpan: 12 },
          { panelId: 'task-queue', col: 2, row: 0, colSpan: 5, rowSpan: 12 },
          { panelId: 'agent-text', col: 7, row: 0, colSpan: 5, rowSpan: 6 },
          { panelId: 'agent-visual', col: 7, row: 6, colSpan: 5, rowSpan: 6 },
        ]
      }
    },
    availablePanels: ['task-queue', 'agent-text', 'file-explorer', 'agent-visual'],
    defaultPanelId: 'task-queue'
  },
  {
    id: 'design',
    name: 'Design',
    icon: '🎨',
    description: 'Visual asset and UI/UX design suite',
    layouts: {
      'semi-full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'agent-visual', col: 0, row: 0, colSpan: 8, rowSpan: 12 },
          { panelId: 'prompt-playground', col: 8, row: 0, colSpan: 4, rowSpan: 12 },
        ]
      },
      'full': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'prompt-playground', col: 0, row: 0, colSpan: 3, rowSpan: 12 },
          { panelId: 'agent-visual', col: 3, row: 0, colSpan: 9, rowSpan: 9 },
          { panelId: 'sandbox-monitor', col: 3, row: 9, colSpan: 9, rowSpan: 3 },
        ]
      },
      'expanded': {
        cols: 12, rows: 12,
        cells: [
          { panelId: 'prompt-playground', col: 0, row: 0, colSpan: 3, rowSpan: 12 },
          { panelId: 'agent-visual', col: 3, row: 0, colSpan: 7, rowSpan: 8 },
          { panelId: 'sandbox-monitor', col: 3, row: 8, colSpan: 7, rowSpan: 4 },
          { panelId: 'telemetry', col: 10, row: 0, colSpan: 2, rowSpan: 4 },
          { panelId: 'schema-inspector', col: 10, row: 4, colSpan: 2, rowSpan: 4 },
          { panelId: 'settings', col: 10, row: 8, colSpan: 2, rowSpan: 4 },
        ]
      }
    },
    availablePanels: ['agent-visual', 'prompt-playground', 'sandbox-monitor'],
    defaultPanelId: 'agent-visual'
  }
];
