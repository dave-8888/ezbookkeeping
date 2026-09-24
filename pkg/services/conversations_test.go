package services

import (
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
	"time"
)

func TestConversationDurableIdempotencyIsolationAndRecovery(t *testing.T) {
	cfg := &settings.Config{DatabaseConfig: &settings.DatabaseConfig{DatabaseType: settings.Sqlite3DbType, DatabasePath: filepath.Join(t.TempDir(), "chat.db")}}
	require.NoError(t, datastore.InitializeDataStore(cfg))
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.ConversationSession), new(models.ConversationTurn)))
	c := core.NewNullContext()
	row, session, fresh, err := Conversations.Begin(c, 1, "chat", "m1", "查询")
	require.NoError(t, err)
	require.True(t, fresh)
	require.Equal(t, "{}", session.State)
	duplicate, _, fresh, err := Conversations.Begin(c, 1, "chat", "m1", "查询")
	require.NoError(t, err)
	require.False(t, fresh)
	require.Equal(t, row.Key, duplicate.Key)
	_, _, _, err = Conversations.Begin(c, 1, "chat", "m1", "不同指令")
	require.ErrorIs(t, err, ErrConversationRequestChanged)
	_, _, _, err = Conversations.Begin(c, 1, "chat", "m2", "更新")
	require.ErrorIs(t, err, ErrConversationBusy)
	other, _, fresh, err := Conversations.Begin(c, 2, "chat", "m1", "其他用户")
	require.NoError(t, err)
	require.True(t, fresh)
	require.NotEqual(t, row.Key, other.Key)
	row.Status = "completed"
	row.State = `{"includeHidden":true}`
	row.Response = `{"status":"completed","reply":"资产已查到"}`
	require.NoError(t, Conversations.Checkpoint(c, row, true))
	next, session, _, err := Conversations.Begin(c, 1, "chat", "m2", "更新")
	require.NoError(t, err)
	require.Contains(t, session.State, "includeHidden")
	next.Status = "writing"
	next.Response = `{"verified":"保留的核实凭据"}`
	require.NoError(t, Conversations.Checkpoint(c, next, false))
	_, err = Conversations.UserDataDB(1).NewSession(c).ID(conversationKey(1, "chat")).Cols("updated_unix_time").Update(&models.ConversationSession{UpdatedUnixTime: time.Now().Unix() - 601})
	require.NoError(t, err)
	require.NoError(t, Conversations.RecoverExpired(c, 1, "chat"))
	recovered, err := Conversations.Get(c, 1, "chat", "m2")
	require.NoError(t, err)
	require.Equal(t, "unknown", recovered.Status)
	require.Contains(t, recovered.Response, "保留的核实凭据")
	_, session, _, err = Conversations.Begin(c, 1, "chat", "m3", "核实")
	require.NoError(t, err)
	require.Contains(t, session.State, "blockedWrite")
	missing, err := Conversations.Get(c, 3, "chat", "m1")
	require.NoError(t, err)
	require.Nil(t, missing)
}
