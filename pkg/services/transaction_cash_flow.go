package services

import (
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/validators"
	"xorm.io/xorm"
)

// prepareCashFlow retains currency, but never attaches income or expenses to an account.
// Legacy clients may still supply an account as a currency hint.
func (s *TransactionService) prepareCashFlow(c core.Context, sess *xorm.Session, transaction *models.Transaction) error {
	if !transaction.IsCashFlow() {
		return nil
	}
	if transaction.Currency == "" && transaction.AccountId > 0 {
		account := &models.Account{}
		has, err := sess.ID(transaction.AccountId).Where("uid=?", transaction.Uid).Get(account)
		if err != nil {
			return err
		}
		if !has {
			return errs.ErrSourceAccountNotFound
		}
		transaction.Currency = account.Currency
	}
	if transaction.Currency == "" {
		user := &models.User{}
		has, err := s.UserDB().NewSession(c).ID(transaction.Uid).Get(user)
		if err != nil {
			return err
		}
		if !has {
			return errs.ErrUserNotFound
		}
		transaction.Currency = user.DefaultCurrency
	}
	if !validators.AllCurrencyNames[transaction.Currency] {
		return errs.ErrAccountCurrencyInvalid
	}
	transaction.AccountId = 0
	transaction.RelatedAccountId = 0
	transaction.RelatedAccountAmount = 0
	return nil
}

// DetachCashFlowAccounts is an idempotent migration. Account balances are intentionally
// untouched: they are independently maintained snapshots, not a sum of cash flow.
func (s *TransactionService) DetachCashFlowAccounts(c core.Context) error {
	for index := 0; index < s.UserDataDBCount(); index++ {
		err := s.UserDataDBByIndex(index).DoTransaction(c, func(sess *xorm.Session) error {
			for {
				var rows []*models.Transaction
				if err := sess.Where("type IN (?,?) AND account_id<>0", models.TRANSACTION_DB_TYPE_INCOME, models.TRANSACTION_DB_TYPE_EXPENSE).Limit(500).Find(&rows); err != nil {
					return err
				}
				if len(rows) == 0 {
					break
				}
				for _, row := range rows {
					if err := s.prepareCashFlow(c, sess, row); err != nil {
						return err
					}
					if _, err := sess.ID(row.TransactionId).Cols("account_id", "related_account_id", "related_account_amount", "currency").Update(row); err != nil {
						return err
					}
				}
			}
			for {
				var templates []*models.TransactionTemplate
				if err := sess.Where("type IN (?,?) AND account_id<>0", models.TRANSACTION_TYPE_INCOME, models.TRANSACTION_TYPE_EXPENSE).Limit(500).Find(&templates); err != nil {
					return err
				}
				if len(templates) == 0 {
					return nil
				}
				for _, template := range templates {
					if err := s.prepareCashFlowTemplate(c, sess, template); err != nil {
						return err
					}
					if _, err := sess.ID(template.TemplateId).Cols("account_id", "related_account_id", "related_account_amount", "currency").Update(template); err != nil {
						return err
					}
				}
			}

		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TransactionService) prepareCashFlowTemplate(c core.Context, sess *xorm.Session, template *models.TransactionTemplate) error {
	dbType, err := template.Type.ToTransactionDbType()
	if err != nil {
		return err
	}
	transaction := &models.Transaction{Uid: template.Uid, Type: dbType, AccountId: template.AccountId, Currency: template.Currency}
	if !transaction.IsCashFlow() {
		return nil
	}
	if err := s.prepareCashFlow(c, sess, transaction); err != nil {
		return err
	}
	template.Currency = transaction.Currency
	template.AccountId = 0
	template.RelatedAccountId = 0
	template.RelatedAccountAmount = 0
	return nil
}
