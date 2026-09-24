package data

import "encoding/json"

// Tool definitions are supplied by the application, never accepted from model output.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ChatMessage struct {
	ReasoningContent *string    `json:"reasoningContent,omitempty"`
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ToolCalls        []ToolCall `json:"toolCalls,omitempty"`
	ToolCallID       string     `json:"toolCallId,omitempty"`
	ToolName         string     `json:"toolName,omitempty"`
	// Preserve Gemini response parts verbatim, including thought signatures.
	NativeParts json.RawMessage `json:"nativeParts,omitempty"`
}
