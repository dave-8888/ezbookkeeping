package core

import "context"

// Context is the base context of financially_free
type Context interface {
	context.Context
	GetContextId() string
	GetClientLocale() string
}
