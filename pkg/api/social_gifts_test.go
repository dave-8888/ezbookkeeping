package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/mayswind/financially_free/pkg/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
)

func TestValidateSocialGiftMetadata(t *testing.T) {
	assert.Nil(t, validateSocialGiftMetadata(models.SOCIAL_GIFT_DIRECTION_SENT, models.SOCIAL_GIFT_EVENT_WEDDING, ""))
	assert.Equal(t, errs.ErrSocialGiftDirectionInvalid, validateSocialGiftMetadata("transfer", models.SOCIAL_GIFT_EVENT_WEDDING, ""))
	assert.Equal(t, errs.ErrSocialGiftEventTypeInvalid, validateSocialGiftMetadata(models.SOCIAL_GIFT_DIRECTION_SENT, "invalid", ""))
	assert.Equal(t, errs.ErrSocialGiftEventNameRequired, validateSocialGiftMetadata(models.SOCIAL_GIFT_DIRECTION_RECEIVED, models.SOCIAL_GIFT_EVENT_OTHER, "  "))
}

func TestSocialGiftRecordCreateRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, binding.Validator.Engine().(*validator.Validate).RegisterValidation("validCurrency", validators.ValidCurrency))
	tests := []struct {
		name      string
		body      string
		wantError bool
	}{
		{name: "valid", body: `{"contactId":"1","direction":"sent","eventType":"wedding","categoryId":"2","time":1700000000,"accountId":"3","amount":100}`},
		{name: "zero amount", body: `{"contactId":"1","direction":"sent","eventType":"wedding","categoryId":"2","time":1700000000,"accountId":"3","amount":0}`, wantError: true},
		{name: "missing account", body: `{"contactId":"1","direction":"sent","eventType":"wedding","categoryId":"2","time":1700000000,"amount":100}`},
		{name: "both empty", body: `{"contactId":"1","direction":"sent","eventType":"wedding","categoryId":"0","time":1700000000,"accountId":"0","amount":100,"currency":"CNY"}`},
		{name: "account only", body: `{"contactId":"1","direction":"sent","eventType":"wedding","time":1700000000,"accountId":"3","amount":100}`},
		{name: "negative account", body: `{"contactId":"1","direction":"sent","eventType":"wedding","time":1700000000,"accountId":"-1","amount":100}`, wantError: true},
		{name: "invalid currency", body: `{"contactId":"1","direction":"sent","eventType":"wedding","time":1700000000,"amount":100,"currency":"BAD"}`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/social_gifts/records/add.json", strings.NewReader(test.body))
			context.Request.Header.Set("Content-Type", "application/json")
			var request models.SocialGiftRecordCreateRequest
			err := context.ShouldBindJSON(&request)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, int64(100), request.Amount)
			}
		})
	}
}

func TestSocialGiftHandlersRequirePrivacyUnlock(t *testing.T) {
	handlers := []core.ApiHandlerFunc{
		SocialGifts.ContactListHandler, SocialGifts.ContactGetHandler,
		SocialGifts.ContactCreateHandler, SocialGifts.ContactModifyHandler, SocialGifts.ContactDeleteHandler,
		SocialGifts.RecordListHandler, SocialGifts.RecordCreateHandler,
		SocialGifts.RecordModifyHandler, SocialGifts.RecordDeleteHandler, SocialGifts.SummaryHandler,
	}
	for _, handler := range handlers {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		result, err := handler(core.WrapWebContext(ctx))
		require.Nil(t, result)
		require.Equal(t, errs.ErrSocialGiftPrivacyLocked, err)
	}
}
