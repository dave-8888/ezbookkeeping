package mcp

import (
	"testing"
	"time"

	"github.com/mayswind/financially_free/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestMCPAddTransactionPreservesLocalDateAndOffset(t *testing.T) {
	for _, value := range []string{"2026-01-01T00:15:00+08:00", "2026-03-08T01:30:00-08:00", "2026-03-08T03:30:00-07:00"} {
		parsed, err := time.Parse(time.RFC3339, value)
		require.NoError(t, err)
		transaction, err := MCPAddTransactionToolHandler.createNewTransactionModel(1, &MCPAddTransactionRequest{Type: "expense", Time: value, Amount: "20"}, 2, 3, 0, "")
		require.NoError(t, err)
		require.Equal(t, utils.GetMinTransactionTimeFromUnixTime(parsed.Unix()), transaction.TransactionTime)
		_, offset := parsed.Zone()
		require.EqualValues(t, offset/60, transaction.TimezoneUtcOffset)
	}
}
