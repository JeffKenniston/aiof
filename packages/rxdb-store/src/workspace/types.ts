import type { ComponentType, LazyExoticComponent } from 'react';

export type PanelId = string;

export interface PanelProps {
  panelId: string;
}

export interface PanelDefinition {
  id: PanelId;
  title: string;
  icon: string;
  category: 'core' | 'power' | 'specialized';
  component: LazyExoticComponent<ComponentType<PanelProps>>;
}

export interface LayoutCell {
  panelId: PanelId;
  row: number;
  col: number;
  rowSpan: number;
  colSpan: number;
}

export interface WorkspaceLayoutPreset {
  cols: number;
  rows: number;
  cells: LayoutCell[];
}

export interface WorkspaceLayout {
  id: string;
  name: string;
  icon: string;
  description: string;
  layouts: {
    'semi-full': WorkspaceLayoutPreset;
    'full': WorkspaceLayoutPreset;
    'expanded': WorkspaceLayoutPreset;
  };
  availablePanels: PanelId[];
  defaultPanelId: PanelId;
}

