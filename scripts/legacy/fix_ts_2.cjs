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
  { regex: /import React, \{ useRef, useEffect, Suspense \} from 'react';/g, replacement: "import React, { useRef, useEffect, Suspense } from 'react';" },
  { regex: /import \{ useCallback \} from 'react';/g, replacement: "import { useCallback } from 'react';" },
  { regex: /import \{ useState, useMemo, useCallback/g, replacement: 'import { useCallback' },
  { regex: /import \{ useState, useCallback/g, replacement: 'import { useCallback' },
  { regex: /import \{ useCallback, useMemo/g, replacement: 'import { useCallback' },
  { regex: /useMemo,/g, replacement: '' }
]);

replaceInFile('src/layout/CommandBar.tsx', [
  { regex: /const \{ toggleCommandBar, setActiveProject, setProjectManagerOpen \} = useLayoutStore\(\)/g, replacement: 'const { toggleCommandBar } = useLayoutStore()' },
  { regex: /setActiveProject, setProjectManagerOpen/g, replacement: '' }
]);

replaceInFile('src/panels/Chat.tsx', [
  { regex: /const modelsList = \['gemini-2\.5-pro', 'gemini-2\.5-flash', 'claude-3-5-sonnet', 'gpt-4o'\];/g, replacement: '' }
]);

replaceInFile('src/panels/Telemetry.tsx', [
  { regex: /const agentsData = useRxQuery<AgentLogDocType\[\]>\(agentLogsQuery as any\);/g, replacement: 'const agentsData = useRxQuery<AgentLogDocType[]>(agentLogsQuery as any) || [];' }
]);

replaceInFile('src/workstations/DataScienceWorkstation.tsx', [
  { regex: /<DatasetSidebar onClose=\{closeSidebar\} \/>/g, replacement: '<DatasetSidebar />' },
  { regex: /<DatasetSidebar onClose=\{\(\) => \{\}\} \/>/g, replacement: '<DatasetSidebar />' },
  { regex: /onClose=\{\(\) => setRightSidebarOpen\(false\)\}/g, replacement: '' }
]);

console.log('TS fixes applied');
