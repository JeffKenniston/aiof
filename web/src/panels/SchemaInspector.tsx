import { useState } from 'react';
import Ajv from 'ajv';

const ajv = new Ajv();

const schema = {
  type: "object",
  properties: {
    "1_reasoning": { type: "array", items: { type: "string" } },
    "2_target_nodes": { type: "array", items: { type: "string" } },
    "3_dynamic_state_mutations": { type: "object" }
  },
  required: ["1_reasoning", "2_target_nodes", "3_dynamic_state_mutations"],
  additionalProperties: false
};

const validateSchema = ajv.compile(schema);

export default function SchemaInspector({  }: { panelId: string }) {
  const [inputJson, setInputJson] = useState('{\n  "1_reasoning": ["Found root cause"],\n  "2_target_nodes": ["executor"],\n  "3_dynamic_state_mutations": {}\n}');
  const [validationResult, setValidationResult] = useState<{valid: boolean, errors?: any}>({ valid: true });

  const handleValidate = () => {
    try {
      const data = JSON.parse(inputJson);
      const valid = validateSchema(data);
      if (valid) {
        setValidationResult({ valid: true });
      } else {
        setValidationResult({ valid: false, errors: validateSchema.errors });
      }
    } catch (err: any) {
      setValidationResult({ valid: false, errors: [{ message: 'Invalid JSON format: ' + err.message }] });
    }
  };

  return (
    <div className="w-full h-full bg-surface-base flex flex-col md:flex-row text-sm">
      <div className="w-full md:w-1/2 p-4 border-r border-border-default overflow-y-auto">
        <h2 className="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
          <svg className="w-5 h-5 text-accent-blue" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
          CognitiveSequence Schema
        </h2>
        
        <div className="bg-surface-raised border border-border-default rounded-lg p-4 font-mono text-xs text-text-secondary space-y-2">
          <div className="flex items-center gap-2">
            <span className="font-bold text-accent-purple">object</span>
          </div>
          <div className="pl-4 border-l border-border-subtle ml-2 space-y-3 pt-2">
            
            <div>
              <div className="flex items-center gap-2 mb-1">
                <span className="text-text-primary font-bold">1_reasoning</span>
                <span className="text-accent-blue bg-accent-blue/10 px-1.5 py-0.5 rounded text-[10px]">array of strings</span>
                <span className="text-accent-red text-[10px] uppercase font-bold border border-accent-red/30 px-1 rounded">Required</span>
              </div>
              <div className="text-text-muted text-[11px] pl-2">Step-by-step chain of thought reasoning.</div>
            </div>

            <div>
              <div className="flex items-center gap-2 mb-1">
                <span className="text-text-primary font-bold">2_target_nodes</span>
                <span className="text-accent-blue bg-accent-blue/10 px-1.5 py-0.5 rounded text-[10px]">array of strings</span>
                <span className="text-accent-red text-[10px] uppercase font-bold border border-accent-red/30 px-1 rounded">Required</span>
              </div>
              <div className="text-text-muted text-[11px] pl-2">Agent node IDs to route to next.</div>
            </div>

            <div>
              <div className="flex items-center gap-2 mb-1">
                <span className="text-text-primary font-bold">3_dynamic_state_mutations</span>
                <span className="text-accent-purple bg-accent-purple/10 px-1.5 py-0.5 rounded text-[10px]">object</span>
                <span className="text-accent-red text-[10px] uppercase font-bold border border-accent-red/30 px-1 rounded">Required</span>
              </div>
              <div className="text-text-muted text-[11px] pl-2">State updates to apply to the orchestrator.</div>
            </div>

          </div>
        </div>
      </div>

      <div className="w-full md:w-1/2 flex flex-col bg-surface-base">
        <div className="p-4 border-b border-border-default bg-surface-raised flex justify-between items-center shrink-0">
          <h3 className="font-semibold text-text-primary">Test Payload</h3>
          <button 
            onClick={handleValidate}
            className="px-4 py-1.5 bg-accent-blue text-white rounded font-medium hover:bg-blue-600 transition-colors shadow-lg shadow-blue-500/20"
          >
            Validate
          </button>
        </div>
        
        <div className="flex-1 p-4 bg-surface-base">
          <textarea
            className="w-full h-full bg-surface-raised border border-border-default rounded p-3 text-text-primary font-mono text-xs resize-none focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue transition-all"
            value={inputJson}
            onChange={e => setInputJson(e.target.value)}
          />
        </div>
        
        <div className={`p-4 border-t border-border-default shrink-0 min-h-[100px] ${validationResult.valid ? 'bg-accent-green/10' : 'bg-accent-red/10'}`}>
          {validationResult.valid ? (
            <div className="flex items-center gap-2 text-accent-green font-medium">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
              Schema Validation Passed
            </div>
          ) : (
            <div className="text-accent-red">
              <div className="flex items-center gap-2 font-medium mb-2">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
                Validation Failed:
              </div>
              <ul className="list-disc pl-5 font-mono text-xs space-y-1">
                {validationResult.errors?.map((err: any, i: number) => (
                  <li key={i}>{err.instancePath} {err.message}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
