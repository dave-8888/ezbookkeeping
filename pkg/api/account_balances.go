package api

import (
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/log"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/services"
	"github.com/mayswind/financially_free/pkg/settings"
)

// AccountBalancesApi represents account balance modification api
type AccountBalancesApi struct {
	ApiUsingConfig
	accounts *services.AccountService
}

// Initialize an account balances api singleton instance
var (
	AccountBalances = &AccountBalancesApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		accounts: services.Accounts,
	}
)

// AccountBalanceModifyRequest represents all parameters of account balance modification request
type AccountBalanceModifyRequest struct {
	Id              string `json:"id" binding:"required"`
	Balance         *int64 `json:"balance" binding:"required"`
	ExpectedBalance *int64 `json:"expectedBalance"`
}

// AccountBatchBalanceModifyRequest represents all parameters of batch account balance modification request
type AccountBatchBalanceModifyRequest struct {
	AccountIds []string `json:"accountIds" binding:"required"`
	Operation  string   `json:"operation" binding:"required,oneof=add subtract"`
	Amount     int64    `json:"amount" binding:"required,min=1"`
}

// ModifySingleAccountBalanceHandler modifies a single account balance
func (a *AccountBalancesApi) ModifySingleAccountBalanceHandler(c *core.WebContext) (any, *errs.Error) {
	var req AccountBalanceModifyRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[account_balances.ModifySingleAccountBalanceHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	newBalance := *req.Balance

	// Allow negative balance for liability accounts like credit cards
	// if newBalance < 0 {
	// 	log.Warnf(c, "[account_balances.ModifySingleAccountBalanceHandler] balance cannot be negative")
	// 	return nil, errs.ErrAmountInvalid
	// }

	uid := c.GetCurrentUid()

	// First get the current account to get currency info
	account, err := a.accounts.GetAccountByAccountId(c, uid, req.Id)

	if err != nil {
		log.Errorf(c, "[account_balances.ModifySingleAccountBalanceHandler] failed to get account \"id:%s\" for user \"uid:%d\", because %s", req.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if account == nil {
		return nil, errs.ErrAccountNotFound
	}

	oldBalance := account.Balance

	// Modify the balance
	if req.ExpectedBalance != nil {
		err = a.accounts.ModifyAccountBalance(c, uid, req.Id, newBalance, *req.ExpectedBalance)
	} else {
		err = a.accounts.ModifyAccountBalance(c, uid, req.Id, newBalance)
	}

	if err != nil {
		log.Errorf(c, "[account_balances.ModifySingleAccountBalanceHandler] failed to modify account \"id:%s\" balance for user \"uid:%d\", because %s", req.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[account_balances.ModifySingleAccountBalanceHandler] user \"uid:%d\" has modified account \"id:%s\" balance from %d to %d",
		uid, req.Id, oldBalance, newBalance)

	return &models.AccountInfoResponse{
		Id:       account.AccountId,
		Name:     account.Name,
		Balance:  newBalance,
		Currency: account.Currency,
	}, nil
}

// BatchModifyAccountBalancesHandler modifies multiple account balances in batch
func (a *AccountBalancesApi) BatchModifyAccountBalancesHandler(c *core.WebContext) (any, *errs.Error) {
	var req AccountBatchBalanceModifyRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[account_balances.BatchModifyAccountBalancesHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	//if req.Amount <= 0 {
	//	log.Warnf(c, "[account_balances.BatchModifyAccountBalancesHandler] amount must be greater than 0")
	//	return nil, errs.ErrAmountInvalid
	//}

	uid := c.GetCurrentUid()

	modifiedAccounts, err := a.accounts.ModifyMultipleAccountBalances(c, uid, req.AccountIds, req.Operation, req.Amount)

	if err != nil {
		log.Errorf(c, "[account_balances.BatchModifyAccountBalancesHandler] failed to modify accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if len(modifiedAccounts) == 0 {
		return nil, errs.ErrAccountNotFound
	}

	response := make([]*models.AccountInfoResponse, len(modifiedAccounts))

	for i, account := range modifiedAccounts {
		response[i] = &models.AccountInfoResponse{
			Id:       account.AccountId,
			Name:     account.Name,
			Balance:  account.Balance,
			Currency: account.Currency,
		}
	}

	log.Infof(c, "[account_balances.BatchModifyAccountBalancesHandler] user \"uid:%d\" has modified %d accounts in batch", uid, len(modifiedAccounts))

	return response, nil
}

// GetAccountBalanceHistoryHandler gets account balance modification history
func (a *AccountBalancesApi) GetAccountBalanceHistoryHandler(c *core.WebContext) (any, *errs.Error) {
	// TODO: Implement account balance modification history
	// This would require adding a new table to store balance modification logs
	return nil, errs.ErrNotImplemented
}
