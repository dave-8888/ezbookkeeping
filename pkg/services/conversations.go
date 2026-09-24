package services

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/models"
	"time"
	"xorm.io/xorm"
)

var ErrConversationBusy = errors.New("conversation has an active request")
var ErrConversationRequestChanged = errors.New("message id already used with different text")
var ErrConversationLeaseLost = errors.New("conversation request no longer active")

type ConversationService struct{ ServiceUsingDB }

var Conversations = &ConversationService{ServiceUsingDB: ServiceUsingDB{container: datastore.Container}}

func conversationKey(uid int64, values ...string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%q", uid, values))))
}

func (s *ConversationService) Get(c core.Context, uid int64, conversation, message string) (*models.ConversationTurn, error) {
	row := &models.ConversationTurn{}
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	ok, err := sess.ID(conversationKey(uid, conversation, message)).Where("uid=?", uid).Get(row)
	if err != nil || !ok {
		return nil, err
	}
	return row, nil
}
func (s *ConversationService) History(c core.Context, uid int64, conversation, exclude string) ([]*models.ConversationTurn, error) {
	var rows []*models.ConversationTurn
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	err := sess.Where("uid=? AND conversation_id=? AND message_id<>?", uid, conversation, exclude).Desc("created_unix_milli", "key").Limit(12).Find(&rows)
	return rows, err
}

// Begin acquires a database conversation lease and inserts the request atomically.
// A repeated message ID returns its stored result, never executes again.
func (s *ConversationService) Begin(c core.Context, uid int64, conversation, message, text string) (*models.ConversationTurn, *models.ConversationSession, bool, error) {
	var row *models.ConversationTurn
	var session *models.ConversationSession
	fresh := false
	err := s.UserDataDB(uid).DoTransaction(c, func(tx *xorm.Session) error {
		row = &models.ConversationTurn{}
		found, err := tx.ID(conversationKey(uid, conversation, message)).Where("uid=?", uid).Get(row)
		if err != nil {
			return err
		}
		if found {
			if row.Text != text {
				return ErrConversationRequestChanged
			}
			return nil
		}
		session = &models.ConversationSession{}
		found, err = tx.ID(conversationKey(uid, conversation)).Where("uid=?", uid).Get(session)
		if err != nil {
			return err
		}
		now := time.Now().Unix()
		if !found {
			session = &models.ConversationSession{Key: conversationKey(uid, conversation), Uid: uid, ConversationId: conversation, State: "{}"}
			if _, err = tx.Insert(session); err != nil {
				return err
			}
		}
		if session.ActiveMessage != "" {
			return ErrConversationBusy
		}
		session.ActiveMessage = message
		session.UpdatedUnixTime = now
		changed, err := tx.ID(session.Key).Where("active_message=?", "").Cols("active_message", "updated_unix_time").Update(session)
		if err != nil {
			return err
		}
		if changed != 1 {
			return ErrConversationBusy
		}
		row = &models.ConversationTurn{Key: conversationKey(uid, conversation, message), Uid: uid, ConversationId: conversation, MessageId: message, Text: text, Status: "processing", State: session.State, CreatedUnixMilli: time.Now().UnixMilli(), UpdatedUnixTime: now}
		_, err = tx.Insert(row)
		fresh = err == nil
		return err
	})
	return row, session, fresh, err
}
func (s *ConversationService) Checkpoint(c core.Context, row *models.ConversationTurn, done bool) error {
	return s.UserDataDB(row.Uid).DoTransaction(c, func(tx *xorm.Session) error {
		now := time.Now().Unix()
		session := &models.ConversationSession{UpdatedUnixTime: now}
		cols := []string{"updated_unix_time"}
		if done {
			session.ActiveMessage = ""
			session.State = row.State
			cols = append(cols, "active_message", "state")
		}
		n, err := tx.ID(conversationKey(row.Uid, row.ConversationId)).Where("uid=? AND active_message=?", row.Uid, row.MessageId).SetExpr("revision", "revision+1").Cols(cols...).Update(session)
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrConversationLeaseLost
		}
		row.UpdatedUnixTime = now
		_, err = tx.ID(row.Key).Where("uid=?", row.Uid).Cols("status", "response", "trace", "state", "updated_unix_time").Update(row)
		return err
	})
}

// Recover only expired requests; active workers have a bounded (<10 minute) lease.
// Recovery never replays model calls or writes.
func (s *ConversationService) RecoverExpired(c core.Context, uid int64, conversation string) error {
	return s.UserDataDB(uid).DoTransaction(c, func(tx *xorm.Session) error {
		session := &models.ConversationSession{}
		ok, err := tx.ID(conversationKey(uid, conversation)).Where("uid=?", uid).Get(session)
		if err != nil || !ok || session.ActiveMessage == "" {
			return err
		}
		if session.UpdatedUnixTime > time.Now().Unix()-600 {
			return nil
		}
		row := &models.ConversationTurn{}
		ok, err = tx.ID(conversationKey(uid, conversation, session.ActiveMessage)).Get(row)
		if err != nil || !ok {
			return err
		}
		// Keep any checkpointed facts, but require a fresh read before another mutation.
		row.Status = "unknown"
		response := map[string]any{}
		_ = json.Unmarshal([]byte(row.Response), &response)
		response["status"] = "unknown"
		oldReply, _ := response["reply"].(string)
		response["reply"] = oldReply + "\n上次处理已中断，请先查询账目或余额核实，系统不会重复提交。"
		raw, _ := json.Marshal(response)
		row.Response = string(raw)
		state := map[string]any{}
		_ = json.Unmarshal([]byte(row.State), &state)
		state["blockedWrite"] = true
		raw, _ = json.Marshal(state)
		row.State = string(raw)
		row.UpdatedUnixTime = time.Now().Unix()
		// Compare the heartbeat revision so recovery cannot overtake a live checkpoint.
		active, revision := session.ActiveMessage, session.Revision
		session.ActiveMessage = ""
		session.State = row.State
		session.UpdatedUnixTime = row.UpdatedUnixTime
		n, err := tx.ID(session.Key).Where("uid=? AND active_message=? AND revision=?", uid, active, revision).
			SetExpr("revision", "revision+1").Cols("active_message", "state", "updated_unix_time").Update(session)
		if err != nil || n != 1 {
			return err
		}
		_, err = tx.ID(row.Key).Cols("status", "response", "state", "updated_unix_time").Update(row)
		return err
	})
}
