package main

import (
	"fmt"
	"aiof/internal/llm"
)

func main() {
	classifier := llm.NewRouteLLMClassifier(nil, "route_llm_weights.json")
	if classifier != nil {
		fmt.Println("Successfully loaded RouteLLM weights!")
	} else {
		fmt.Println("Failed to load classifier")
	}
}
