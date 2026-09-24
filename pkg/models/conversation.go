package models

// ConversationSession keeps model/tool context on the server, scoped to its owner.
type ConversationSession struct {
	Key             string `xorm:"PK VARCHAR(64)"`
	Uid             int64  `xorm:"NOT NULL INDEX"`
	ConversationId  string `xorm:"VARCHAR(64) NOT NULL"`
	ActiveMessage   string `xorm:"VARCHAR(64) NOT NULL"`
	State           string `xorm:"MEDIUMTEXT"`
	UpdatedUnixTime int64
	Revision        int64 `xorm:"NOT NULL DEFAULT 0"`
}

// ConversationTurn is also the durable idempotency record for one user message.
type ConversationTurn struct {
	Key              string `xorm:"PK VARCHAR(64)"`
	Uid              int64  `xorm:"NOT NULL INDEX(conversation_turn_owner)"`
	ConversationId   string `xorm:"VARCHAR(64) NOT NULL INDEX(conversation_turn_owner)"`
	MessageId        string `xorm:"VARCHAR(64) NOT NULL"`
	Text             string `xorm:"TEXT"`
	Status           string `xorm:"VARCHAR(32)"`
	Response         string `xorm:"MEDIUMTEXT"`
	Trace            string `xorm:"MEDIUMTEXT"`
	State            string `xorm:"MEDIUMTEXT"`
	CreatedUnixMilli int64
	UpdatedUnixTime  int64
}
