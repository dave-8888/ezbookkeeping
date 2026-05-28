import { CategoryType } from '@/core/category.ts';
import { TransactionType } from '@/core/transaction.ts';
import { Account } from '@/models/account.ts';
import { TransactionCategory } from '@/models/transaction_category.ts';
import { TransactionTag } from '@/models/transaction_tag.ts';
import { TransactionPicture } from '@/models/transaction_picture_info.ts';
import { Transaction } from '@/models/transaction.ts';
import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

import {
    isDefined,
    isNumber
} from './common.ts';
import {
    getTimezoneOffsetMinutes,
    getSameDateTimeWithCurrentTimezone,
    parseDateTimeFromUnixTimeWithBrowserTimezone,
    getCurrentUnixTime
} from './datetime.ts';
import {
    categoryTypeToTransactionType,
    isSubCategoryIdAvailable,
    getFirstVisibleCategoryId,
    getFirstAvailableSubCategoryId
} from './category.ts';

export interface SetTransactionOptions {
    time?: number;
    type?: number;
    categoryId?: string;
    accountId?: string;
    destinationAccountId?: string;
    amount?: number;
    destinationAmount?: number;
    tagIds?: string;
    comment?: string;
}

export interface RecognizedTransactionCreateContext {
    readonly allCategories: Record<number, TransactionCategory[]>;
    readonly allCategoriesMap: Record<string, TransactionCategory>;
    readonly allVisibleAccounts: Account[];
    readonly allAccountsMap: Record<string, Account>;
    readonly allTagsMap: Record<string, TransactionTag>;
    readonly defaultAccountId: string;
    readonly timeZone: string;
}

export function createTransactionFromRecognizedResult(result: RecognizedTransactionResponse, context: RecognizedTransactionCreateContext): Transaction {
    const transactionTime = result.time ?? getCurrentUnixTimeForNewTransaction();
    const transactionType = getRecognizedTransactionType(result, context.allCategoriesMap) ?? TransactionType.Expense;
    const transaction = Transaction.createNewTransaction(
        transactionType,
        transactionTime,
        context.timeZone,
        getTimezoneOffsetMinutes(transactionTime, context.timeZone)
    );

    setTransactionModelByTransaction(
        transaction,
        null,
        context.allCategories,
        context.allCategoriesMap,
        context.allVisibleAccounts,
        context.allAccountsMap,
        context.allTagsMap,
        context.defaultAccountId,
        {
            time: result.time,
            type: getRecognizedTransactionType(result, context.allCategoriesMap),
            categoryId: result.categoryId,
            accountId: result.sourceAccountId,
            destinationAccountId: result.destinationAccountId,
            amount: result.sourceAmount,
            destinationAmount: result.destinationAmount,
            tagIds: result.tagIds ? result.tagIds.join(',') : undefined,
            comment: result.comment
        },
        false
    );

    fillRecognizedTransferDestinationAmount(transaction, result, context.allAccountsMap);

    return transaction;
}

export function getRecognizedTransactionAutoCreateMissingFields(result: RecognizedTransactionResponse, transaction: Transaction, context: RecognizedTransactionCreateContext): string[] {
    const missingFields: string[] = [];

    if (!getRecognizedTransactionType(result, context.allCategoriesMap)) {
        missingFields.push('type');
    }

    if (!transaction.getCategoryId()) {
        missingFields.push('categoryId');
    }

    if (result.categoryId && !missingFields.includes('categoryId') && !isRecognizedCategoryCompatibleWithTransactionType(result, transaction.type, context.allCategoriesMap)) {
        missingFields.push('categoryId');
    }

    if (!transaction.sourceAccountId) {
        missingFields.push('sourceAccountId');
    }

    if (!isNumber(result.sourceAmount) || result.sourceAmount === 0) {
        missingFields.push('sourceAmount');
    }

    if (transaction.type === TransactionType.Transfer) {
        if (!transaction.destinationAccountId || transaction.sourceAccountId === transaction.destinationAccountId) {
            missingFields.push('destinationAccountId');
        }

        if (!isNumber(transaction.destinationAmount) || transaction.destinationAmount <= 0) {
            missingFields.push('destinationAmount');
        }

        if (transaction.sourceAmount < 0) {
            missingFields.push('sourceAmount');
        }
    }

    return missingFields;
}

export function setTransactionModelByTransaction(transaction: Transaction, transaction2: Transaction | null | undefined, allCategories: Record<number, TransactionCategory[]>, allCategoriesMap: Record<string, TransactionCategory>, allVisibleAccounts: Account[], allAccountsMap: Record<string, Account>, allTagsMap: Record<string, TransactionTag>, defaultAccountId: string, options: SetTransactionOptions, setContextData: boolean): void {
    if (isDefined(options.time)) {
        transaction.time = options.time;
        transaction.utcOffset = getTimezoneOffsetMinutes(transaction.time, transaction.timeZone);
    }

    if (!options.type && options.categoryId && options.categoryId !== '0' && allCategoriesMap[options.categoryId]) {
        const category = allCategoriesMap[options.categoryId] as TransactionCategory;
        const type = categoryTypeToTransactionType(category.type);

        if (isNumber(type)) {
            transaction.type = type;
        }
    }

    if (isDefined(options.amount)) {
        transaction.sourceAmount = options.amount;
    }

    if (isDefined(options.destinationAmount)) {
        transaction.destinationAmount = options.destinationAmount;
    }

    if (allCategories[CategoryType.Expense] &&
        allCategories[CategoryType.Expense].length) {
        if (options.categoryId && options.categoryId !== '0') {
            if (isSubCategoryIdAvailable(allCategories[CategoryType.Expense], options.categoryId)) {
                transaction.expenseCategoryId = options.categoryId;
            } else {
                transaction.expenseCategoryId = getFirstAvailableSubCategoryId(allCategories[CategoryType.Expense], options.categoryId);
            }
        }

        if (!transaction.expenseCategoryId) {
            transaction.expenseCategoryId = getFirstVisibleCategoryId(allCategories[CategoryType.Expense]);
        }
    }

    if (allCategories[CategoryType.Income] &&
        allCategories[CategoryType.Income].length) {
        if (options.categoryId && options.categoryId !== '0') {
            if (isSubCategoryIdAvailable(allCategories[CategoryType.Income], options.categoryId)) {
                transaction.incomeCategoryId = options.categoryId;
            } else {
                transaction.incomeCategoryId = getFirstAvailableSubCategoryId(allCategories[CategoryType.Income], options.categoryId);
            }
        }

        if (!transaction.incomeCategoryId) {
            transaction.incomeCategoryId = getFirstVisibleCategoryId(allCategories[CategoryType.Income]);
        }
    }

    if (allCategories[CategoryType.Transfer] &&
        allCategories[CategoryType.Transfer].length) {
        if (options.categoryId && options.categoryId !== '0') {
            if (isSubCategoryIdAvailable(allCategories[CategoryType.Transfer], options.categoryId)) {
                transaction.transferCategoryId = options.categoryId;
            } else {
                transaction.transferCategoryId = getFirstAvailableSubCategoryId(allCategories[CategoryType.Transfer], options.categoryId);
            }
        }

        if (!transaction.transferCategoryId) {
            transaction.transferCategoryId = getFirstVisibleCategoryId(allCategories[CategoryType.Transfer]);
        }
    }

    if (allVisibleAccounts.length) {
        if (options.accountId && options.accountId !== '0') {
            for (const account of allVisibleAccounts) {
                if (account.id === options.accountId) {
                    transaction.sourceAccountId = options.accountId;
                    transaction.destinationAccountId = options.accountId;
                    break;
                }
            }
        }

        if (options.destinationAccountId && options.destinationAccountId !== '0') {
            for (const account of allVisibleAccounts) {
                if (account.id === options.destinationAccountId) {
                    transaction.destinationAccountId = options.destinationAccountId;
                    break;
                }
            }
        }

        if (!transaction.sourceAccountId) {
            if (defaultAccountId && allAccountsMap[defaultAccountId] && !allAccountsMap[defaultAccountId].hidden) {
                transaction.sourceAccountId = defaultAccountId;
            } else {
                transaction.sourceAccountId = allVisibleAccounts[0]!.id;
            }
        }

        if (!transaction.destinationAccountId) {
            if (defaultAccountId && allAccountsMap[defaultAccountId] && !allAccountsMap[defaultAccountId].hidden) {
                transaction.destinationAccountId = defaultAccountId;
            } else {
                transaction.destinationAccountId = allVisibleAccounts[0]!.id;
            }
        }
    }

    if (allTagsMap && options.tagIds) {
        const tagIds = options.tagIds.split(',');
        const finalTagIds = [];

        for (const tagId of tagIds) {
            const tag = allTagsMap[tagId];

            if (tag && !tag.hidden) {
                finalTagIds.push(tag.id);
            }
        }

        transaction.tagIds = finalTagIds;
    }

    if (options.comment) {
        transaction.comment = options.comment;
    }

    if (transaction2) {
        if (setContextData) {
            transaction.id = transaction2.id;
        }

        transaction.type = transaction2.type;

        if (transaction.type === TransactionType.Expense) {
            transaction.expenseCategoryId = transaction2.categoryId || '';
        } else if (transaction.type === TransactionType.Income) {
            transaction.incomeCategoryId = transaction2.categoryId || '';
        } else if (transaction.type === TransactionType.Transfer) {
            transaction.transferCategoryId = transaction2.categoryId || '';
        }

        if (setContextData) {
            transaction.time = transaction2.time;
            transaction.timeZone = transaction2.timeZone;
            transaction.utcOffset = transaction2.utcOffset;
        }

        transaction.sourceAccountId = transaction2.sourceAccountId;

        if (transaction2.destinationAccountId) {
            transaction.destinationAccountId = transaction2.destinationAccountId;
        } else {
            transaction.destinationAccountId = '';
        }

        transaction.sourceAmount = transaction2.sourceAmount;

        if (transaction2.destinationAmount) {
            transaction.destinationAmount = transaction2.destinationAmount;
        } else {
            transaction.destinationAmount = 0;
        }

        transaction.hideAmount = transaction2.hideAmount;
        transaction.tagIds = transaction2.tagIds || [];
        transaction.setPictures(TransactionPicture.ofMulti(transaction2.pictures || []));

        transaction.comment = transaction2.comment;

        if (setContextData) {
            transaction.setGeoLocation(transaction2.geoLocation);
        }
    }
}

function getRecognizedTransactionType(result: RecognizedTransactionResponse, allCategoriesMap: Record<string, TransactionCategory>): TransactionType | undefined {
    if (result.type === TransactionType.Income || result.type === TransactionType.Expense || result.type === TransactionType.Transfer) {
        return result.type;
    }

    if (result.categoryId && allCategoriesMap[result.categoryId]) {
        const category = allCategoriesMap[result.categoryId] as TransactionCategory;
        const transactionType = categoryTypeToTransactionType(category.type);

        if (transactionType === TransactionType.Income || transactionType === TransactionType.Expense || transactionType === TransactionType.Transfer) {
            return transactionType;
        }
    }

    return undefined;
}

function isRecognizedCategoryCompatibleWithTransactionType(result: RecognizedTransactionResponse, transactionType: number, allCategoriesMap: Record<string, TransactionCategory>): boolean {
    if (!result.categoryId || !allCategoriesMap[result.categoryId]) {
        return false;
    }

    const category = allCategoriesMap[result.categoryId] as TransactionCategory;
    return categoryTypeToTransactionType(category.type) === transactionType;
}

function fillRecognizedTransferDestinationAmount(transaction: Transaction, result: RecognizedTransactionResponse, allAccountsMap: Record<string, Account>): void {
    if (transaction.type !== TransactionType.Transfer || isDefined(result.destinationAmount) || !isNumber(result.sourceAmount)) {
        return;
    }

    const sourceAccount = allAccountsMap[transaction.sourceAccountId];
    const destinationAccount = allAccountsMap[transaction.destinationAccountId];

    if (!sourceAccount || !destinationAccount || sourceAccount.currency === destinationAccount.currency) {
        transaction.destinationAmount = transaction.sourceAmount;
    }
}

function getCurrentUnixTimeForNewTransaction(): number {
    return getSameDateTimeWithCurrentTimezone(parseDateTimeFromUnixTimeWithBrowserTimezone(getCurrentUnixTime())).getUnixTime();
}
