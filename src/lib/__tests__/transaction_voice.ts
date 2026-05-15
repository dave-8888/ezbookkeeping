import { describe, expect, test } from '@jest/globals';

import { AccountCategory, AccountType } from '@/core/account.ts';
import { CategoryType } from '@/core/category.ts';
import { TransactionType } from '@/core/transaction.ts';

import { Account } from '@/models/account.ts';
import { TransactionCategory } from '@/models/transaction_category.ts';
import { TransactionTag } from '@/models/transaction_tag.ts';

import { parseTransactionTextLocally } from '@/lib/transaction_voice.ts';

describe('parseTransactionTextLocally', () => {
    test('parses spoken expense amount, account, category, tag and relative date', () => {
        const result = parseTransactionTextLocally('昨天午饭 38 元 支付宝 餐饮 工作', {
            allCategories: createCategories(),
            allVisibleAccounts: createAccounts(),
            allTags: createTags(),
            currentUnixTime: Math.floor(new Date(2026, 4, 7, 12, 0, 0).getTime() / 1000)
        });

        expect(result.type).toBe(TransactionType.Expense);
        expect(result.sourceAmount).toBe(3800);
        expect(result.sourceAccountId).toBe('account_alipay');
        expect(result.categoryId).toBe('category_lunch');
        expect(result.tagIds).toStrictEqual(['tag_work']);
        expect(result.time).toBe(Math.floor(new Date(2026, 4, 6, 0, 0, 0).getTime() / 1000));
        expect(result.comment).toBe('昨天午饭 38 元 支付宝 餐饮 工作');
    });

    test('parses income keywords and exact time', () => {
        const result = parseTransactionTextLocally('今天 13:20 工资 1000 招商银行', {
            allCategories: createCategories(),
            allVisibleAccounts: createAccounts(),
            allTags: createTags(),
            currentUnixTime: Math.floor(new Date(2026, 4, 7, 12, 0, 0).getTime() / 1000)
        });

        expect(result.type).toBe(TransactionType.Income);
        expect(result.sourceAmount).toBe(100000);
        expect(result.sourceAccountId).toBe('account_cmb');
        expect(result.categoryId).toBe('category_salary');
        expect(result.time).toBe(Math.floor(new Date(2026, 4, 7, 13, 20, 0).getTime() / 1000));
    });

    test('uses longest account and category matches', () => {
        const result = parseTransactionTextLocally('支付宝 12.34 餐饮', {
            allCategories: createCategories(),
            allVisibleAccounts: createAccounts(),
            allTags: createTags()
        });

        expect(result.sourceAccountId).toBe('account_alipay');
        expect(result.categoryId).toBe('category_lunch');
        expect(result.sourceAmount).toBe(1234);
    });

    test('keeps original text as comment when no structured fields match', () => {
        const result = parseTransactionTextLocally('随便记一下', {
            allCategories: createCategories(),
            allVisibleAccounts: createAccounts(),
            allTags: createTags()
        });

        expect(result).toStrictEqual({
            comment: '随便记一下'
        });
    });
});

function createAccounts(): Account[] {
    return [
        Account.of({
            id: 'account_ali',
            name: '支付',
            parentId: '0',
            category: AccountCategory.VirtualAccount.type,
            type: AccountType.SingleAccount.type,
            icon: '1',
            color: '000000',
            currency: 'CNY',
            balance: 0,
            comment: '',
            displayOrder: 1,
            hidden: false
        }),
        Account.of({
            id: 'account_alipay',
            name: '支付宝',
            parentId: '0',
            category: AccountCategory.VirtualAccount.type,
            type: AccountType.SingleAccount.type,
            icon: '1',
            color: '000000',
            currency: 'CNY',
            balance: 0,
            comment: '',
            displayOrder: 2,
            hidden: false
        }),
        Account.of({
            id: 'account_cmb',
            name: '招商银行',
            parentId: '0',
            category: AccountCategory.CheckingAccount.type,
            type: AccountType.SingleAccount.type,
            icon: '1',
            color: '000000',
            currency: 'CNY',
            balance: 0,
            comment: '',
            displayOrder: 3,
            hidden: false
        })
    ];
}

function createCategories(): Record<number, TransactionCategory[]> {
    return {
        [CategoryType.Expense]: [
            TransactionCategory.of({
                id: 'category_food',
                name: '餐饮',
                parentId: '0',
                type: CategoryType.Expense,
                icon: '1',
                color: '000000',
                comment: '',
                displayOrder: 1,
                hidden: false,
                subCategories: [{
                    id: 'category_lunch',
                    name: '午饭',
                    parentId: 'category_food',
                    type: CategoryType.Expense,
                    icon: '1',
                    color: '000000',
                    comment: '',
                    displayOrder: 1,
                    hidden: false
                }]
            })
        ],
        [CategoryType.Income]: [
            TransactionCategory.of({
                id: 'category_income',
                name: '收入',
                parentId: '0',
                type: CategoryType.Income,
                icon: '1',
                color: '000000',
                comment: '',
                displayOrder: 1,
                hidden: false,
                subCategories: [{
                    id: 'category_salary',
                    name: '工资',
                    parentId: 'category_income',
                    type: CategoryType.Income,
                    icon: '1',
                    color: '000000',
                    comment: '',
                    displayOrder: 1,
                    hidden: false
                }]
            })
        ],
        [CategoryType.Transfer]: []
    };
}

function createTags(): TransactionTag[] {
    return [
        TransactionTag.of({
            id: 'tag_work',
            name: '工作',
            displayOrder: 1,
            hidden: false
        })
    ];
}
