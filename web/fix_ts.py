import os
import glob
import re

files = glob.glob('src/panels/*.tsx')
for f in files:
    with open(f, 'r') as file:
        content = file.read()
    
    # 1. Fix React import
    content = re.sub(r"import React, {", "import {", content)
    content = re.sub(r"import React from 'react';\n", "", content)
    
    # 2. Fix type imports from db/index
    content = re.sub(r"import { getDatabase, (\w+DocType) }", r"import { getDatabase, type \1 }", content)
    
    # 3. Fix type imports from @xyflow/react
    content = re.sub(r"(\s+)Node,(\s+)Edge", r"\1type Node,\2type Edge", content)
    content = re.sub(r"(\s+)NodeProps,", r"\1type NodeProps,", content)
    
    # 4. Fix Background opacity
    content = content.replace('opacity={0.05}', 'style={{ opacity: 0.05 }}')
    
    # 5. Fix unused panelId
    content = content.replace('{ panelId }: { panelId: string }', '{  }: { panelId: string }')
    content = content.replace('{ panelId }: AgentVisualProps', '{  }: AgentVisualProps')
    
    with open(f, 'w') as file:
        file.write(content)

print("Fixed TS errors")
