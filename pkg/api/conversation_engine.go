package api

import (
	"encoding/json"
	"errors"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"strings"
	"time"
)

// Only this public result crosses the mobile API boundary. Provider messages,
// thoughts, signatures, tool calls and execution state stay in the service.
type conversationReply struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Reply     string `json:"reply"`
	Details   string `json:"details,omitempty"`
	Verified  string `json:"verified,omitempty"`
}
type serverConversationEngine struct {
	messages   []data.ChatMessage
	trace      []data.ChatMessage
	state      *conversationState
	output     conversationReply
	model      func([]data.ChatMessage) (*data.ChatMessage, error)
	execute    func(string, json.RawMessage) (conversationToolResult, error)
	checkpoint func(string) error
	diagnose   func(string, error)
}

func (e *serverConversationEngine) run(text string) conversationReply {
	e.trace = append(e.trace, data.ChatMessage{Role: "user", Content: text})
	e.output.Status = "completed"
	deadline := time.Now().Add(8 * time.Minute)
	for step := 0; step < 8; step++ {
		if time.Now().After(deadline) {
			e.fail("deadline", errors.New("conversation deadline reached"))
			return e.output
		}
		input := append(append([]data.ChatMessage{}, e.messages...), e.trace...)
		response, err := e.model(input)
		if err != nil || response == nil {
			e.fail("model", err)
			return e.output
		}
		if len(response.ToolCalls) == 0 {
			if strings.TrimSpace(response.Content) == "" {
				e.fail("empty_response", errors.New("empty model response"))
				return e.output
			}
			e.trace = append(e.trace, *response)
			e.output.Reply = response.Content
			return e.output
		}
		if len(response.ToolCalls) > 4 {
			e.fail("tool_limit", errors.New("too many tool calls"))
			return e.output
		}
		e.trace = append(e.trace, *response)
		unknown := false
		for _, call := range response.ToolCalls {
			var result conversationToolResult
			if unknown {
				result.Data = map[string]any{"status": "error", "message": "写入结果待核实，本轮停止执行"}
			} else {
				result, err = e.execute(call.Name, call.Arguments)
				if err != nil {
					e.diagnose("tool_"+call.Name, err)
					result.Data = map[string]any{"status": "error", "message": "操作未完成，请核对参数或向用户澄清，不要假装成功。", "reason": err.Error()}
				}
			}
			if result.Reply != "" {
				e.output.Reply = result.Reply
			}
			if result.Details != "" {
				e.output.Details = result.Details
			}
			if result.Verified != "" {
				if e.output.Verified != "" {
					e.output.Verified += "\n"
				}
				e.output.Verified += result.Verified
			}
			if result.Unknown {
				unknown = true
				e.output.Status = "unknown"
			}
			raw, marshalErr := json.Marshal(result.Data)
			if marshalErr != nil {
				e.fail("tool_serialization", marshalErr)
				return e.output
			}
			e.trace = append(e.trace, data.ChatMessage{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: string(raw)})
			// Tool results and verified writes are durable before the next model request.
			if err = e.checkpoint("processing"); err != nil {
				e.fail("checkpoint", err)
				e.output.Status = "unknown"
				return e.output
			}
		}
		if unknown {
			return e.output
		}
	}
	e.fail("step_limit", errors.New("conversation step limit"))
	return e.output
}
func (e *serverConversationEngine) fail(stage string, err error) {
	e.diagnose(stage, err)
	// A model summarization failure must not replace already retrieved real data.
	if e.output.Reply != "" {
		if e.output.Verified != "" && !strings.Contains(e.output.Reply, e.output.Verified) {
			e.output.Reply = e.output.Verified + "\n" + e.output.Reply
		}
		return
	}
	e.output.Status = "interrupted"
	e.output.Reply = "这次暂时没能处理完成，请稍后再试。"
}
