package agent

import (
	"encoding/json"

	"google.golang.org/genai"
)

// CognitiveSequence enforces ADR-05 Lexicographical Property Ordering for CoT.
// We prefix JSON keys to mathematically guarantee the LLM outputs reasoning steps 
// before predicting target nodes or tools, ensuring proper Chain-of-Thought execution.
type CognitiveSequence struct {
	// 1_reasoning forces the step-by-step cognitive deduction process first.
	Reasoning []string `json:"1_reasoning"`
	
	// 2_target_nodes represents the Set-Valued Predictor routing (ADR-03).
	// Facilitates concurrent agent dispatches instead of Top-1 bottlenecks.
	TargetNodes []string `json:"2_target_nodes"`
	
	// 3_dynamic_state_mutations defers reflection using raw bytes (ADR-04 Two-Pass Deserialization).
	DynamicStateMutations json.RawMessage `json:"3_dynamic_state_mutations"`

	// 4_chat_reply holds the conversational reply.
	ChatReply string `json:"4_chat_reply"`
}

// BuildRoutingSchema constructs the strict OpenAPI schema for the Gemini Gateway.
// It complies with Phase 2.1 (Two-Pass Deserialization constraints).
func BuildRoutingSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"1_reasoning": {
				Type: genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
				Description: "Step-by-step cognitive deduction process. Must be evaluated first.",
			},
			"2_target_nodes": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
					Enum: []string{
						"fabricator_agent", "code_developer_agent", "qa_tester_agent", "research_agent", "architecture_agent",
						"devops_agent", "secops_agent", "database_agent", "ui_designer_agent", "doc_agent",
						"sre_agent", "chaos_agent", "mlops_agent", "product_owner_agent", "localization_agent",
						"performance_agent", "mobile_agent", "data_agent", "a11y_agent", "redteam_agent",
						"scrum_master_agent", "legal_agent", "devex_agent", "finops_agent", "hardware_agent",
						"web3_agent", "graphics_agent", "support_agent", "quantum_agent", "sandbox_orchestrator",
					},
				},
				Description: "Array of target agent nodes to dispatch in parallel (Set-Valued).",
			},
			"3_dynamic_state_mutations": {
				Type: genai.TypeObject,
				Description: "Flat state mutations (Max depth: 2).",
			},
			"4_chat_reply": {
				Type:        genai.TypeString,
				Description: "Conversational reply to the user based on your persona and the current Workstation View Mode.",
			},
		},
		Required: []string{"1_reasoning", "2_target_nodes", "3_dynamic_state_mutations", "4_chat_reply"},
		PropertyOrdering: []string{"1_reasoning", "2_target_nodes", "3_dynamic_state_mutations", "4_chat_reply"},
	}
}
