const fs = require('fs');
const path = require('path');

function replaceInFile(filePath, regexReplacements) {
  const fullPath = path.join(__dirname, filePath);
  if (!fs.existsSync(fullPath)) return;
  
  let content = fs.readFileSync(fullPath, 'utf8');
  for (const { regex, replacement } of regexReplacements) {
    content = content.replace(regex, replacement);
  }
  fs.writeFileSync(fullPath, content);
}

replaceInFile('src/components/SpatialCanvas.tsx', [
  { regex: /import \{ useState, useMemo, useCallback/g, replacement: 'import { useCallback' },
  { regex: /import \{ useState, useCallback/g, replacement: 'import { useCallback' },
  { regex: /const \[nodes, setNodes, onNodesChange\]/g, replacement: 'const [nodes, , onNodesChange]' },
  { regex: /import React, \{ useState, useRef, useEffect, useMemo, Suspense \} from 'react';/g, replacement: "import React, { useRef, useEffect, Suspense } from 'react';" }
]);

replaceInFile('src/layout/CommandBar.tsx', [
  { regex: /const \{ toggleCommandBar, setActiveProject, setProjectManagerOpen \} = useLayoutStore\(\)/g, replacement: 'const { toggleCommandBar } = useLayoutStore()' },
]);

replaceInFile('src/panels/AgentTextLog.tsx', [
  { regex: /const logs = useRxQuery<AgentLogDocType\[\]>\(query as any\);/g, replacement: 'const logs = useRxQuery<AgentLogDocType[]>(query as any) || [];' },
]);

replaceInFile('src/panels/Chat.tsx', [
  { regex: /const modelsList = \['gemini-2\.5-pro', 'gemini-2\.5-flash', 'claude-3-5-sonnet', 'gpt-4o'\];/g, replacement: '' }
]);

replaceInFile('src/panels/ContextCache.tsx', [
  { regex: /const caches = useRxQuery<ContextCacheDocType\[\]>\(query as any\);/g, replacement: 'const caches = useRxQuery<ContextCacheDocType[]>(query as any) || [];' }
]);

replaceInFile('src/panels/KnowledgeGraph.tsx', [
  { regex: /export default function KnowledgeGraph\(\{ panelId \}: \{ panelId\?: string \}\) \{/g, replacement: 'export default function KnowledgeGraph() {' }
]);

replaceInFile('src/panels/SandboxMonitor.tsx', [
  { regex: /const containers = useRxQuery<SandboxStateDocType\[\]>\(query as any\);/g, replacement: 'const containers = useRxQuery<SandboxStateDocType[]>(query as any) || [];' }
]);

replaceInFile('src/panels/TaskQueue.tsx', [
  { regex: /const tasks = useRxQuery<TaskQueueDocType\[\]>\(query as any\);/g, replacement: 'const tasks = useRxQuery<TaskQueueDocType[]>(query as any) || [];' }
]);

replaceInFile('src/panels/Telemetry.tsx', [
  { regex: /const data = useRxQuery<TelemetryDocType\[\]>\(query as any\);/g, replacement: 'const data = useRxQuery<TelemetryDocType[]>(query as any) || [];' },
  { regex: /const agentsData = useRxQuery<AgentLogDocType\[\]>\(agentLogsQuery as any\);/g, replacement: 'const agentsData = useRxQuery<AgentLogDocType[]>(agentLogsQuery as any) || [];' }
]);

replaceInFile('src/panels/WALInspector.tsx', [
  { regex: /const events = useRxQuery<WALEventDocType\[\]>\(query as any\);/g, replacement: 'const events = useRxQuery<WALEventDocType[]>(query as any) || [];' }
]);

replaceInFile('src/workstations/DataScienceWorkstation.tsx', [
  { regex: /<DatasetSidebar onClose=\{closeSidebar\} \/>/g, replacement: '<DatasetSidebar />' }
]);

replaceInFile('src/workstations/EducationWorkstation.tsx', [
  { regex: /import Chat from '\.\.\/panels\/Chat';/g, replacement: '' }
]);

replaceInFile('src/workstations/FabricatorWorkstation.tsx', [
  { regex: /import \{ useState, use, Suspense, useMemo \} from 'react';/g, replacement: "import { useState, Suspense } from 'react';" },
  { regex: /import \{ getDatabase \} from '\.\.\/db\/index';/g, replacement: '' },
  { regex: /import \{ useRxQuery \} from '\.\.\/hooks\/useRxQuery';/g, replacement: '' },
  { regex: /const db = use\(getDatabase\(\)\);/g, replacement: '' },
  { regex: /const activeProject = useLayoutStore\(\(state\) => state\.activeProject\);/g, replacement: '' }
]);

replaceInFile('src/workstations/ProfessionalWorkstation.tsx', [
  { regex: /<KPIWidget\s+label="Tasks Completed"\s+value="1,248"\s+trend="\+12%"\s+icon="clock"\s+\/>/g, replacement: '<KPIWidget label="Tasks Completed" collectionName="task_queue" icon="clock" />' },
  { regex: /<KPIWidget\s+label="System Messages"\s+value="8,492"\s+trend="\+5%"\s+icon="mail"\s+\/>/g, replacement: '<KPIWidget label="System Messages" collectionName="agent_logs" icon="mail" />' }
]);

console.log('TS fixes applied');
