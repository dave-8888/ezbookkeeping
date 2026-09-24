import { describe, expect, test } from '@jest/globals';
import { TransactionType } from '@/core/transaction.ts';
import { Transaction } from '@/models/transaction.ts';
import { TransactionTemplate } from '@/models/transaction_template.ts';

describe('independent cash flow', () => {
    test.each([TransactionType.Income, TransactionType.Expense])('persists currency and removes account links for %s', type => {
        const transaction = Transaction.createNewTransaction(type, 1750000000, 'UTC', 0);
        transaction.currency = 'USD';
        transaction.sourceAccountId = '123';
        transaction.destinationAccountId = '456';
        expect(transaction.toCreateRequest('test')).toMatchObject({ currency: 'USD', sourceAccountId: '0', destinationAccountId: '0' });
        expect(transaction.toModifyRequest()).toMatchObject({ currency: 'USD', sourceAccountId: '0' });
        const draft = Transaction.ofDraft(transaction.toTransactionDraft());
        expect(draft?.currency).toBe('USD');
        const template = TransactionTemplate.createNewTransactionTemplate(transaction);
        expect(template.toTemplateCreateRequest('test')).toMatchObject({ currency: 'USD', sourceAccountId: '0' });
    });
    test('transfers keep account links', () => {
        const transaction = Transaction.createNewTransaction(TransactionType.Transfer, 1750000000, 'UTC', 0);
        transaction.sourceAccountId = '123';
        transaction.destinationAccountId = '456';
        expect(transaction.toCreateRequest('test')).toMatchObject({ sourceAccountId: '123', destinationAccountId: '456' });
    });
});
