package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountBalanceModifyRequestAllowsZeroBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts/balance/modify.json", strings.NewReader(`{"id":"1","balance":0}`))
	ginContext.Request.Header.Set("Content-Type", "application/json")

	var request AccountBalanceModifyRequest
	err := ginContext.ShouldBindJSON(&request)

	require.NoError(t, err)
	require.NotNil(t, request.Balance)
	assert.Equal(t, int64(0), *request.Balance)
}

func TestAccountBalanceModifyRequestRequiresBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts/balance/modify.json", strings.NewReader(`{"id":"1"}`))
	ginContext.Request.Header.Set("Content-Type", "application/json")

	var request AccountBalanceModifyRequest
	err := ginContext.ShouldBindJSON(&request)

	require.Error(t, err)
	assert.Nil(t, request.Balance)
}
