package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestParseRecognizedTransactionResponse(t *testing.T) {
	api := &LargeLanguageModelsApi{}
	clientTimezone := time.UTC
	recognitionContext := createTestTransactionRecognitionContext()

	response, err := api.parseRecognizedTransactionResponse(core.NewNullContext(), 1, clientTimezone, &models.RecognizedTransactionResult{
		Type:                   "transfer",
		Time:                   "2024-01-02 03:04:05",
		Amount:                 "12.34",
		DestinationAmount:      "10",
		AccountName:            "Cash",
		DestinationAccountName: "Bank",
		CategoryName:           "Transfer",
		TagNames:               []string{"Work", "Hidden"},
		Description:            "voice text",
	}, recognitionContext)

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, models.TRANSACTION_TYPE_TRANSFER, response.Type)
	assert.Equal(t, time.Date(2024, 1, 2, 3, 4, 5, 0, clientTimezone).Unix(), response.Time)
	assert.Equal(t, int64(3001), response.CategoryId)
	assert.Equal(t, int64(1001), response.SourceAccountId)
	assert.Equal(t, int64(1002), response.DestinationAccountId)
	assert.Equal(t, int64(1234), response.SourceAmount)
	assert.Equal(t, int64(1000), response.DestinationAmount)
	assert.Equal(t, []string{"4001"}, response.TagIds)
	assert.Equal(t, "voice text", response.Comment)
}

func TestParseRecognizedTransactionResponseEmptyResult(t *testing.T) {
	api := &LargeLanguageModelsApi{}

	response, err := api.parseRecognizedTransactionResponse(core.NewNullContext(), 1, time.UTC, &models.RecognizedTransactionResult{}, createTestTransactionRecognitionContext())

	assert.Nil(t, response)
	assert.Equal(t, errs.ErrNoTransactionInformationInImage, err)
}

func TestParseRecognizedTransactionResponseInvalidType(t *testing.T) {
	api := &LargeLanguageModelsApi{}

	response, err := api.parseRecognizedTransactionResponse(core.NewNullContext(), 1, time.UTC, &models.RecognizedTransactionResult{
		Type: "unknown",
	}, createTestTransactionRecognitionContext())

	assert.Nil(t, response)
	assert.Equal(t, errs.ErrOperationFailed, err)
}

func TestParseRecognizedTransactionResponseInvalidAmount(t *testing.T) {
	api := &LargeLanguageModelsApi{}

	response, err := api.parseRecognizedTransactionResponse(core.NewNullContext(), 1, time.UTC, &models.RecognizedTransactionResult{
		Type:   "expense",
		Amount: "1.234",
	}, createTestTransactionRecognitionContext())

	assert.Nil(t, response)
	assert.Equal(t, errs.ErrOperationFailed, err)
}

func createTestTransactionRecognitionContext() *transactionRecognitionContext {
	return &transactionRecognitionContext{
		accountMap: map[string]*models.Account{
			"Cash": {
				AccountId: 1001,
				Name:      "Cash",
			},
			"Bank": {
				AccountId: 1002,
				Name:      "Bank",
			},
		},
		expenseCategoryMap: map[string]*models.TransactionCategory{
			"Food": {
				CategoryId: 2001,
				Name:       "Food",
				Type:       models.CATEGORY_TYPE_EXPENSE,
			},
		},
		incomeCategoryMap: map[string]*models.TransactionCategory{
			"Salary": {
				CategoryId: 2002,
				Name:       "Salary",
				Type:       models.CATEGORY_TYPE_INCOME,
			},
		},
		transferCategoryMap: map[string]*models.TransactionCategory{
			"Transfer": {
				CategoryId: 3001,
				Name:       "Transfer",
				Type:       models.CATEGORY_TYPE_TRANSFER,
			},
		},
		tagMap: map[string]*models.TransactionTag{
			"Work": {
				TagId: 4001,
				Name:  "Work",
			},
		},
	}
}
