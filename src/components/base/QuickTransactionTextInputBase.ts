import { ref } from 'vue';

import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionTagsStore } from '@/stores/transactionTag.ts';
import { useTransactionsStore } from '@/stores/transaction.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';

import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

import { createTransactionFromRecognizedResult, getRecognizedTransactionAutoCreateMissingFields } from '@/lib/transaction.ts';
import {
    hasRecognizedTransactionFields,
    recognizeTransactionByText
} from '@/lib/transaction_text.ts';
import { generateRandomUUID } from '@/lib/misc.ts';

export interface QuickTransactionTextInputOptions {
    showMessage: (message: string) => void;
    showError: (error: unknown) => void;
    onNeedManualInput: (result: RecognizedTransactionResponse) => void | Promise<void>;
    onSaved?: () => void;
}

export function useQuickTransactionTextInput(options: QuickTransactionTextInputOptions) {
    const accountsStore = useAccountsStore();
    const transactionCategoriesStore = useTransactionCategoriesStore();
    const transactionTagsStore = useTransactionTagsStore();
    const transactionsStore = useTransactionsStore();
    const settingsStore = useSettingsStore();
    const userStore = useUserStore();

    const transactionText = ref<string>('');
    const recognizingByText = ref<boolean>(false);

    function clearTransactionText(): void {
        transactionText.value = '';
    }

    function loadRecognitionContext(): Promise<void> {
        return Promise.all([
            accountsStore.loadAllAccounts({ force: false }),
            transactionCategoriesStore.loadAllCategories({ force: false }),
            transactionTagsStore.loadAllTags({ force: false })
        ]).then(() => undefined);
    }

    function saveRecognizedTransactionOrRequestManualInput(result: RecognizedTransactionResponse): Promise<void> {
        const createContext = {
            allCategories: transactionCategoriesStore.allTransactionCategories,
            allCategoriesMap: transactionCategoriesStore.allTransactionCategoriesMap,
            allVisibleAccounts: accountsStore.allVisiblePlainAccounts,
            allAccountsMap: accountsStore.allAccountsMap,
            allTagsMap: transactionTagsStore.allTransactionTagsMap,
            defaultAccountId: userStore.currentUserDefaultAccountId,
            timeZone: settingsStore.appSettings.timeZone
        };
        const transaction = createTransactionFromRecognizedResult(result, createContext);
        const missingFields = getRecognizedTransactionAutoCreateMissingFields(result, transaction, createContext);

        if (missingFields.length > 0) {
            return Promise.resolve(options.onNeedManualInput(result));
        }

        return transactionsStore.saveTransaction({
            transaction,
            defaultCurrency: userStore.currentUserDefaultCurrency,
            isEdit: false,
            clientSessionId: generateRandomUUID()
        }).then(() => {
            clearTransactionText();
            options.showMessage('You have added a new transaction');
            options.onSaved?.();
        });
    }

    function submitTransactionText(): Promise<void> {
        if (recognizingByText.value) {
            return Promise.resolve();
        }

        const text = transactionText.value.trim();

        if (!text) {
            options.showError('Parameter Invalid');
            return Promise.resolve();
        }

        recognizingByText.value = true;

        return loadRecognitionContext().then(() => {
            return recognizeTransactionByText(text, {
                allCategories: transactionCategoriesStore.allTransactionCategories,
                allVisibleAccounts: accountsStore.allVisiblePlainAccounts,
                allTags: transactionTagsStore.allTransactionTags,
                recognizeTransactionText: (text: string) => transactionsStore.recognizeTransactionText({ text })
            });
        }).then(result => {
            if (!hasRecognizedTransactionFields(result)) {
                options.showError('Unable to recognize transaction text');
                return;
            }

            return saveRecognizedTransactionOrRequestManualInput(result);
        }).catch(error => {
            options.showError(error);
        }).finally(() => {
            recognizingByText.value = false;
        });
    }

    return {
        transactionText,
        recognizingByText,
        clearTransactionText,
        submitTransactionText
    };
}
