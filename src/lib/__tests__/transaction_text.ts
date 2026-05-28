import { describe, expect, test } from '@jest/globals';

import { AccountCategory, AccountType } from '@/core/account.ts';
import { CategoryType } from '@/core/category.ts';
import { TransactionType } from '@/core/transaction.ts';

import { Account } from '@/models/account.ts';
import { TransactionCategory } from '@/models/transaction_category.ts';
import { TransactionTag } from '@/models/transaction_tag.ts';

import {
    hasRecognizedTransactionFields,
    parseTransactionTextLocally,
    recognizeTransactionByText
} from '@/lib/transaction_text.ts';
import {
    createTransactionFromRecognizedResult,
    getRecognizedTransactionAutoCreateMissingFields
} from '@/lib/transaction.ts';

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

    test('parses transfer type and both accounts', () => {
        const result = parseTransactionTextLocally('转账 200 元 支付宝 招商银行', {
            allCategories: createCategories(),
            allVisibleAccounts: createAccounts(),
            allTags: createTags()
        });

        expect(result.type).toBe(TransactionType.Transfer);
        expect(result.sourceAmount).toBe(20000);
        expect(result.sourceAccountId).toBe('account_alipay');
        expect(result.destinationAccountId).toBe('account_cmb');
        expect(result.categoryId).toBe('category_transfer_general');
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
        expect(hasRecognizedTransactionFields(result)).toBe(false);
    });
});

describe('recognizeTransactionByText', () => {
    test('uses LLM result and fills original text as comment', async () => {
        const result = await recognizeTransactionByText('午饭 30 元', {
            allCategories: createCategories(),
            allVisibleAccounts: createAccounts(),
            allTags: createTags(),
            recognizeTransactionText: async () => ({
                type: TransactionType.Expense,
                sourceAmount: 3000
            })
        });

        expect(result).toStrictEqual({
            type: TransactionType.Expense,
            sourceAmount: 3000,
            comment: '午饭 30 元'
        });
        expect(hasRecognizedTransactionFields(result)).toBe(true);
    });

    test('falls back to local parsing when LLM recognition fails', async () => {
        const result = await recognizeTransactionByText('工资 100 招商银行', {
            allCategories: createCategories(),
            allVisibleAccounts: createAccounts(),
            allTags: createTags(),
            recognizeTransactionText: async () => {
                throw new Error('disabled');
            }
        });

        expect(result.type).toBe(TransactionType.Income);
        expect(result.sourceAmount).toBe(10000);
        expect(result.sourceAccountId).toBe('account_cmb');
    });
});

describe('createTransactionFromRecognizedResult', () => {
    test('creates a complete expense transaction from recognized fields', () => {
        const categories = createCategories();
        const accounts = createAccounts();
        const context = createRecognizedTransactionCreateContext(categories, accounts);
        const transaction = createTransactionFromRecognizedResult({
            type: TransactionType.Expense,
            categoryId: 'category_lunch',
            sourceAccountId: 'account_alipay',
            sourceAmount: 3800,
            comment: '午饭 38 元 支付宝'
        }, context);

        expect(transaction.type).toBe(TransactionType.Expense);
        expect(transaction.categoryId).toBe('category_lunch');
        expect(transaction.sourceAccountId).toBe('account_alipay');
        expect(transaction.sourceAmount).toBe(3800);
        expect(getRecognizedTransactionAutoCreateMissingFields({
            type: TransactionType.Expense,
            categoryId: 'category_lunch',
            sourceAccountId: 'account_alipay',
            sourceAmount: 3800
        }, transaction, context)).toStrictEqual([]);
    });

    test('fills transfer destination amount for same-currency accounts', () => {
        const categories = createCategories();
        const accounts = createAccounts();
        const context = createRecognizedTransactionCreateContext(categories, accounts);
        const result = {
            type: TransactionType.Transfer,
            categoryId: 'category_transfer_general',
            sourceAccountId: 'account_alipay',
            destinationAccountId: 'account_cmb',
            sourceAmount: 20000
        };
        const transaction = createTransactionFromRecognizedResult(result, context);

        expect(transaction.type).toBe(TransactionType.Transfer);
        expect(transaction.destinationAmount).toBe(20000);
        expect(getRecognizedTransactionAutoCreateMissingFields(result, transaction, context)).toStrictEqual([]);
    });

    test('requires a recognized transaction type before auto creation', () => {
        const categories = createCategories();
        const accounts = createAccounts();
        const context = createRecognizedTransactionCreateContext(categories, accounts);
        const result = {
            sourceAmount: 1200
        };
        const transaction = createTransactionFromRecognizedResult(result, context);

        expect(getRecognizedTransactionAutoCreateMissingFields(result, transaction, context)).toContain('type');
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
        [CategoryType.Transfer]: [
            TransactionCategory.of({
                id: 'category_transfer',
                name: '转账',
                parentId: '0',
                type: CategoryType.Transfer,
                icon: '1',
                color: '000000',
                comment: '',
                displayOrder: 1,
                hidden: false,
                subCategories: [{
                    id: 'category_transfer_general',
                    name: '转账',
                    parentId: 'category_transfer',
                    type: CategoryType.Transfer,
                    icon: '1',
                    color: '000000',
                    comment: '',
                    displayOrder: 1,
                    hidden: false
                }]
            })
        ]
    };
}

function createRecognizedTransactionCreateContext(categories: Record<number, TransactionCategory[]>, accounts: Account[]) {
    return {
        allCategories: categories,
        allCategoriesMap: createCategoriesMap(categories),
        allVisibleAccounts: accounts,
        allAccountsMap: createAccountsMap(accounts),
        allTagsMap: createTagsMap(createTags()),
        defaultAccountId: 'account_cmb',
        timeZone: 'Asia/Shanghai'
    };
}

function createAccountsMap(accounts: Account[]): Record<string, Account> {
    const accountsMap: Record<string, Account> = {};

    for (const account of accounts) {
        accountsMap[account.id] = account;
    }

    return accountsMap;
}

function createCategoriesMap(categories: Record<number, TransactionCategory[]>): Record<string, TransactionCategory> {
    const categoriesMap: Record<string, TransactionCategory> = {};

    for (const categoryList of Object.values(categories)) {
        for (const category of categoryList) {
            categoriesMap[category.id] = category;

            if (category.subCategories) {
                for (const subCategory of category.subCategories) {
                    categoriesMap[subCategory.id] = subCategory;
                }
            }
        }
    }

    return categoriesMap;
}

function createTagsMap(tags: TransactionTag[]): Record<string, TransactionTag> {
    const tagsMap: Record<string, TransactionTag> = {};

    for (const tag of tags) {
        tagsMap[tag.id] = tag;
    }

    return tagsMap;
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
