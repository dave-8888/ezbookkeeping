package api

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/mayswind/financially_free/pkg/mcp"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/stretchr/testify/require"
)

func transactionMCPCall() data.ToolCall {
	return data.ToolCall{ID: "add", Name: "add_transaction", Arguments: json.RawMessage(`{"type":"expense","time":"2026-01-01T00:15:00+08:00","account_name":"现金","category_name":"餐饮","amount":"20"}`)}
}

func TestMCPTransactionReadsThenWritesOnce(t *testing.T) {
	models, writes, reads, checkpoints := 0, 0, 0, 0
	result := runMCPTransaction("昨天午饭20元", func(messages []data.ChatMessage) (*data.ChatMessage, error) {
		models++
		if models == 1 {
			return &data.ChatMessage{Role: "assistant", ToolCalls: []data.ToolCall{{ID: "accounts", Name: "query_all_accounts", Arguments: json.RawMessage(`{}`)}}}, nil
		}
		require.Equal(t, "tool", messages[len(messages)-1].Role)
		require.Contains(t, messages[len(messages)-1].Content, "现金")
		return &data.ChatMessage{Role: "assistant", ToolCalls: []data.ToolCall{transactionMCPCall()}}, nil
	}, func(call data.ToolCall) (any, error) {
		if call.Name == "query_all_accounts" {
			reads++
			return map[string]string{"account": "现金"}, nil
		}
		require.Equal(t, 1, checkpoints)
		writes++
		return mcp.MCPCallToolResponse[mcp.MCPTextContent]{Content: []*mcp.MCPTextContent{mcp.NewMCPTextContent(`{"success":true}`)}}, nil
	}, func() error { checkpoints++; return nil })
	require.Equal(t, "created", result.Status)
	require.Equal(t, 2, models)
	require.Equal(t, 1, reads)
	require.Equal(t, 1, writes)
	require.Contains(t, result.Reply, "2026-01-01T00:15:00+08:00")
}

func TestMCPTransactionDoesNotTrustProseOrRetryWrites(t *testing.T) {
	for _, test := range []struct {
		name   string
		calls  []data.ToolCall
		err    error
		result any
		status string
		writes int
	}{
		{name: "prose cannot claim success", status: "needs_input"},
		{name: "multiple writes rejected before execution", calls: []data.ToolCall{transactionMCPCall(), transactionMCPCall()}, status: "needs_input"},
		{name: "unknown tool rejected", calls: []data.ToolCall{{Name: "delete_transaction"}}, status: "failed"},
		{name: "uncertain write never retried", calls: []data.ToolCall{transactionMCPCall()}, err: errors.New("connection lost"), status: "unknown", writes: 1},
		{name: "dry run result is not success", calls: []data.ToolCall{transactionMCPCall()}, result: mcp.MCPCallToolResponse[mcp.MCPTextContent]{Content: []*mcp.MCPTextContent{mcp.NewMCPTextContent(`{"success":true,"dry_run":true}`)}}, status: "unknown", writes: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			models, writes := 0, 0
			result := runMCPTransaction("测试", func([]data.ChatMessage) (*data.ChatMessage, error) {
				models++
				return &data.ChatMessage{Content: "已记账", ToolCalls: test.calls}, nil
			}, func(data.ToolCall) (any, error) { writes++; return test.result, test.err }, func() error { return nil })
			require.Equal(t, test.status, result.Status)
			require.Equal(t, test.writes, writes)
			require.Equal(t, 1, models)
		})
	}
}

func TestMCPTransactionRejectsMissingTimezoneAndLostCheckpoint(t *testing.T) {
	for _, timestamp := range []string{"2026-01-01T10:00:00", "2026-02-30T10:00:00+08:00", "2026-01-01T10:00:00+08:00"} {
		call := transactionMCPCall()
		var args map[string]any
		require.NoError(t, json.Unmarshal(call.Arguments, &args))
		args["time"] = timestamp
		call.Arguments, _ = json.Marshal(args)
		result := runMCPTransaction("测试", func([]data.ChatMessage) (*data.ChatMessage, error) {
			return &data.ChatMessage{ToolCalls: []data.ToolCall{call}}, nil
		}, func(data.ToolCall) (any, error) { t.Fatal("must not write"); return nil, nil }, func() error { return errors.New("lost lease") })
		require.NotEqual(t, "created", result.Status)
	}
}

func TestMCPTransactionPromptUsesLocalCalendar(t *testing.T) {
	zone, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2025, 12, 31, 16, 15, 0, 0, time.UTC).In(zone)
	prompt := mcpTransactionPrompt(now, `"现金"`)
	require.Contains(t, prompt, "2026-01-01T00:15:00+08:00")
	require.Contains(t, prompt, "今天：2026-01-01；昨天：2025-12-31；前天：2025-12-30")
	require.Contains(t, prompt, "Thursday")
	zone, err = time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	prompt = mcpTransactionPrompt(time.Date(2026, 3, 9, 7, 15, 0, 0, time.UTC).In(zone), `""`)
	require.Contains(t, prompt, "2026-03-09T00:15:00-07:00")
	require.Contains(t, prompt, "昨天：2026-03-08")
}

func TestMCPTransactionToolsUseRegisteredSchemas(t *testing.T) {
	previous := mcp.Container
	t.Cleanup(func() { mcp.Container = previous })
	require.NoError(t, mcp.InitializeMCPHandlers(&settings.Config{}))
	tools, err := mcpTransactionTools()
	require.NoError(t, err)
	require.Len(t, tools, 4)
	for _, tool := range tools {
		require.True(t, mcpTransactionToolAllowed(tool.Name))
		if tool.Name == "add_transaction" {
			require.Contains(t, string(tool.Parameters), "RFC 3339")
		}
	}
}
