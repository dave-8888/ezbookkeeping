package models

const (
	SOCIAL_GIFT_DIRECTION_SENT     = "sent"
	SOCIAL_GIFT_DIRECTION_RECEIVED = "received"
)

// SocialGiftImportMetadata represents optional social gift data attached to an imported transaction.
type SocialGiftImportMetadata struct {
	ContactKey   string `json:"contactKey" binding:"max=128"`
	ContactName  string `json:"contactName" binding:"required,max=64"`
	Relationship string `json:"relationship" binding:"max=64"`
	ContactNote  string `json:"contactNote" binding:"max=255"`
	EventType    string `json:"eventType" binding:"required,max=32"`
	EventName    string `json:"eventName" binding:"max=64"`
}

const (
	SOCIAL_GIFT_EVENT_WEDDING      = "wedding"
	SOCIAL_GIFT_EVENT_NEWBORN      = "newborn"
	SOCIAL_GIFT_EVENT_BIRTHDAY     = "birthday"
	SOCIAL_GIFT_EVENT_HOUSEWARMING = "housewarming"
	SOCIAL_GIFT_EVENT_EDUCATION    = "education"
	SOCIAL_GIFT_EVENT_FUNERAL      = "funeral"
	SOCIAL_GIFT_EVENT_OTHER        = "other"
)

const (
	SOCIAL_GIFT_STATUS_RECEIVABLE = "receivable"
	SOCIAL_GIFT_STATUS_PAYABLE    = "payable"
	SOCIAL_GIFT_STATUS_SETTLED    = "settled"
)

// SocialGiftContact represents a person in the social-gift ledger.
type SocialGiftContact struct {
	ContactId       int64  `xorm:"PK"`
	Uid             int64  `xorm:"INDEX(IDX_social_gift_contact_uid_deleted) NOT NULL"`
	Deleted         bool   `xorm:"INDEX(IDX_social_gift_contact_uid_deleted) NOT NULL"`
	Name            string `xorm:"VARCHAR(64) NOT NULL"`
	Relationship    string `xorm:"VARCHAR(64) NOT NULL"`
	Comment         string `xorm:"VARCHAR(255) NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// SocialGiftRecord stores an independent gift or links to an accounting transaction.
// Unlinked records use -RecordId as TransactionId to preserve the unique link index.
type SocialGiftRecord struct {
	RecordId          int64  `xorm:"PK"`
	Uid               int64  `xorm:"INDEX(IDX_social_gift_record_uid_deleted_contact) UNIQUE(UQE_social_gift_record_uid_transaction) NOT NULL"`
	Deleted           bool   `xorm:"INDEX(IDX_social_gift_record_uid_deleted_contact) NOT NULL"`
	ContactId         int64  `xorm:"INDEX(IDX_social_gift_record_uid_deleted_contact) NOT NULL"`
	TransactionId     int64  `xorm:"UNIQUE(UQE_social_gift_record_uid_transaction) NOT NULL"`
	EventType         string `xorm:"VARCHAR(32) NOT NULL"`
	EventName         string `xorm:"VARCHAR(64) NOT NULL"`
	Direction         string `xorm:"VARCHAR(16)"`
	Amount            int64
	TransactionTime   int64
	TimezoneUtcOffset int16
	AccountId         int64
	CategoryId        int64
	Currency          string `xorm:"VARCHAR(3)"`
	Comment           string `xorm:"VARCHAR(255)"`
	CreatedUnixTime   int64
	UpdatedUnixTime   int64
	DeletedUnixTime   int64
}

type SocialGiftContactCreateRequest struct {
	Name         string `json:"name" binding:"required,notBlank,max=64"`
	Relationship string `json:"relationship" binding:"max=64"`
	Comment      string `json:"comment" binding:"max=255"`
}

type SocialGiftContactModifyRequest struct {
	Id           int64  `json:"id,string" binding:"required,min=1"`
	Name         string `json:"name" binding:"required,notBlank,max=64"`
	Relationship string `json:"relationship" binding:"max=64"`
	Comment      string `json:"comment" binding:"max=255"`
}

type SocialGiftContactDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

type SocialGiftContactListRequest struct {
	Keyword string `form:"keyword" binding:"max=64"`
	Status  string `form:"status"`
	Page    int32  `form:"page" binding:"min=1"`
	Count   int32  `form:"count" binding:"required,min=1,max=50"`
}

type SocialGiftRecordListRequest struct {
	ContactId int64  `form:"contact_id" binding:"required,min=1"`
	Direction string `form:"direction"`
	EventType string `form:"event_type"`
	Page      int32  `form:"page" binding:"min=1"`
	Count     int32  `form:"count" binding:"required,min=1,max=50"`
}

type SocialGiftRecordCreateRequest struct {
	Currency        string                          `json:"currency" binding:"omitempty,len=3,validCurrency"`
	ContactId       int64                           `json:"contactId,string" binding:"min=0"`
	NewContact      *SocialGiftContactCreateRequest `json:"newContact"`
	Direction       string                          `json:"direction" binding:"required"`
	EventType       string                          `json:"eventType" binding:"required,max=32"`
	EventName       string                          `json:"eventName" binding:"max=64"`
	CategoryId      int64                           `json:"categoryId,string" binding:"min=0"`
	Time            int64                           `json:"time" binding:"required,min=1"`
	UtcOffset       int16                           `json:"utcOffset" binding:"min=-720,max=840"`
	AccountId       int64                           `json:"accountId,string" binding:"min=0"`
	Amount          int64                           `json:"amount" binding:"required,min=1,max=99999999999"`
	Comment         string                          `json:"comment" binding:"max=255"`
	ClientSessionId string                          `json:"clientSessionId"`
}

type SocialGiftRecordModifyRequest struct {
	Currency   string `json:"currency" binding:"omitempty,len=3,validCurrency"`
	Id         int64  `json:"id,string" binding:"required,min=1"`
	ContactId  int64  `json:"contactId,string" binding:"required,min=1"`
	Direction  string `json:"direction" binding:"required"`
	EventType  string `json:"eventType" binding:"required,max=32"`
	EventName  string `json:"eventName" binding:"max=64"`
	CategoryId int64  `json:"categoryId,string" binding:"min=0"`
	Time       int64  `json:"time" binding:"required,min=1"`
	UtcOffset  int16  `json:"utcOffset" binding:"min=-720,max=840"`
	AccountId  int64  `json:"accountId,string" binding:"min=0"`
	Amount     int64  `json:"amount" binding:"required,min=1,max=99999999999"`
	Comment    string `json:"comment" binding:"max=255"`
}

type SocialGiftRecordDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

type SocialGiftBalanceResponse struct {
	Currency       string `json:"currency"`
	SentAmount     int64  `json:"sentAmount"`
	ReceivedAmount int64  `json:"receivedAmount"`
	NetAmount      int64  `json:"netAmount"`
	Status         string `json:"status"`
}

type SocialGiftContactInfoResponse struct {
	Id             int64                        `json:"id,string"`
	Name           string                       `json:"name"`
	Relationship   string                       `json:"relationship"`
	Comment        string                       `json:"comment"`
	RecordCount    int64                        `json:"recordCount"`
	LastRecordTime int64                        `json:"lastRecordTime"`
	Balances       []*SocialGiftBalanceResponse `json:"balances"`
}

type SocialGiftContactPageResponse struct {
	Items      []*SocialGiftContactInfoResponse `json:"items"`
	TotalCount int64                            `json:"totalCount"`
}

type SocialGiftRecordInfoResponse struct {
	Id          int64                    `json:"id,string"`
	ContactId   int64                    `json:"contactId,string"`
	Direction   string                   `json:"direction"`
	EventType   string                   `json:"eventType"`
	EventName   string                   `json:"eventName"`
	Currency    string                   `json:"currency"`
	Transaction *TransactionInfoResponse `json:"transaction"`
}

type SocialGiftRecordPageResponse struct {
	Items      []*SocialGiftRecordInfoResponse `json:"items"`
	TotalCount int64                           `json:"totalCount"`
}

type SocialGiftSummaryResponse struct {
	Balances               []*SocialGiftBalanceResponse `json:"balances"`
	ReceivableContactCount int64                        `json:"receivableContactCount"`
	PayableContactCount    int64                        `json:"payableContactCount"`
}
