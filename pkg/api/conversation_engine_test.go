package api

import (
	"encoding/json"
	"errors"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestServerConversationRetainsQueryWhenFinalModelFails(t *testing.T) {
	calls := 0
	executions := 0
	checkpoints := 0
	e := &serverConversationEngine{state: &conversationState{}, checkpoint: func(string) error { checkpoints++; return nil }, diagnose: func(string, error) {}}
	e.model = func(messages []data.ChatMessage) (*data.ChatMessage, error) {
		calls++
		if calls == 1 {
			return &data.ChatMessage{Role: "assistant", ToolCalls: []data.ToolCall{{ID: "a", Name: "list_accounts", Arguments: json.RawMessage(`{}`)}}}, nil
		}
		require.Equal(t, "tool", messages[len(messages)-1].Role)
		return nil, errors.New("provider protocol failure")
	}
	e.execute = func(name string, args json.RawMessage) (conversationToolResult, error) {
		executions++
		return conversationToolResult{Data: map[string]any{"accounts": []string{"钱包"}}, Reply: "当前资产 37.42 CNY", Details: "1. 钱包\n37.42 CNY"}, nil
	}
	result := e.run("输出我的所有资产")
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "当前资产 37.42 CNY", result.Reply)
	require.Contains(t, result.Details, "钱包")
	require.Equal(t, 1, executions)
	require.Equal(t, 1, checkpoints)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "toolCalls")
	require.NotContains(t, string(raw), "provider")
}
func TestServerConversationMultiStepAndUnknownWriteStops(t *testing.T) {
	steps := 0
	executed := []string{}
	e := &serverConversationEngine{state: &conversationState{}, checkpoint: func(string) error { return nil }, diagnose: func(string, error) {}}
	e.model = func(messages []data.ChatMessage) (*data.ChatMessage, error) {
		steps++
		name := "list_accounts"
		if steps == 2 {
			name = "set_account_balance"
		}
		return &data.ChatMessage{Role: "assistant", ToolCalls: []data.ToolCall{{ID: name, Name: name, Arguments: json.RawMessage(`{}`)}}}, nil
	}
	e.execute = func(name string, args json.RawMessage) (conversationToolResult, error) {
		executed = append(executed, name)
		if name == "list_accounts" {
			return conversationToolResult{Data: map[string]any{"status": "success"}, Reply: "真实资产"}, nil
		}
		return conversationToolResult{Data: map[string]any{"status": "unknown"}, Reply: "结果待核实", Unknown: true}, nil
	}
	result := e.run("看看资产，把第一项改成六千")
	require.Equal(t, []string{"list_accounts", "set_account_balance"}, executed)
	require.Equal(t, 2, steps)
	require.Equal(t, "unknown", result.Status)
}
func TestServerConversationCanClarifyWithoutTools(t *testing.T) {
	e := &serverConversationEngine{state: &conversationState{}, diagnose: func(string, error) {}, model: func([]data.ChatMessage) (*data.ChatMessage, error) {
		return &data.ChatMessage{Role: "assistant", Content: "你要修改哪个账户？"}, nil
	}, execute: func(string, json.RawMessage) (conversationToolResult, error) {
		t.Fatal("must not execute")
		return conversationToolResult{}, nil
	}}
	require.Equal(t, "你要修改哪个账户？", e.run("改一下").Reply)
}
