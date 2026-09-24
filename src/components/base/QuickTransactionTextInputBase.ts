import { computed, ref, watch } from 'vue';

import { useUserLLMSettingsStore } from '@/stores/userLLMSetting.ts';

import { useTransactionsStore } from '@/stores/transaction.ts';
import { useUserStore } from '@/stores/user.ts';
import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';
import { generateRandomUUID } from '@/lib/misc.ts';

export interface QuickTransactionTextInputOptions {
    showMessage: (message: string) => void;
    showError: (error: unknown) => void;
    onNeedManualInput: (result: RecognizedTransactionResponse) => void | Promise<void>;
    onSaved?: () => void;
}

export function useQuickTransactionTextInput(options: QuickTransactionTextInputOptions) {
    const transactionsStore = useTransactionsStore();
    const userStore = useUserStore();

    const userLLMSettingsStore = useUserLLMSettingsStore();
    const aiEnabled = computed<boolean>(() => userLLMSettingsStore.transactionAIEnabled);

    watch(() => userStore.currentUserBasicInfo?.username, username => {
        if (username) {
            // Failed settings loads keep personal AI disabled; the store logs the error.
            userLLMSettingsStore.loadUserLLMSetting({ force: false }).catch(() => undefined);
        }
    }, { immediate: true });

    const transactionText = ref<string>('');
    const recognizingByText = ref<boolean>(false);

    function clearTransactionText(): void {
        transactionText.value = '';
    }

    // Retain the identity across retries, panel remounts and page reloads. The
    // backend durably returns the original result instead of executing twice.
    function pendingRequest(text: string): { text: string, requestId: string } {
        const key = `web-mcp-transaction:${userStore.currentUserBasicInfo?.username}`;
        const raw = sessionStorage.getItem(key);
        const pending: { text: string, requestId: string }[] = raw ? JSON.parse(raw) : [];
        const saved = pending.find(request => request.text === text);

        if (saved?.text === text && saved.requestId) {
            return saved;
        }

        const request = { text, requestId: generateRandomUUID() };
        pending.push(request);
        sessionStorage.setItem(key, JSON.stringify(pending));
        return request;
    }

    function forgetPendingRequest(text: string): void {
        const key = `web-mcp-transaction:${userStore.currentUserBasicInfo?.username}`;
        const raw = sessionStorage.getItem(key);
        const pending: { text: string, requestId: string }[] = raw ? JSON.parse(raw) : [];
        sessionStorage.setItem(key, JSON.stringify(pending.filter(request => request.text !== text)));
    }

    function submitTransactionText(): Promise<void> {
        if (recognizingByText.value) {
            return Promise.resolve();
        }

        if (!aiEnabled.value) {
            options.showError('llm provider is not enabled');
            return Promise.resolve();
        }

        const text = transactionText.value.trim();

        if (!text) {
            options.showError('Parameter Invalid');
            return Promise.resolve();
        }

        recognizingByText.value = true;

        return Promise.resolve().then(() => {
            return transactionsStore.createTransactionByMCP(pendingRequest(text));
        }).then(result => {
            if (result.status === 'created') {
                forgetPendingRequest(text);
                clearTransactionText();
                options.showMessage(result.reply);
                options.onSaved?.();
            } else {
                if (result.status === 'failed' || result.status === 'needs_input') {
                    forgetPendingRequest(text);
                }

                options.showMessage(result.reply);
            }
        }).catch(error => {
            options.showError(error);
        }).finally(() => {
            recognizingByText.value = false;
        });
    }

    return {
        aiEnabled,
        transactionText,
        recognizingByText,
        clearTransactionText,
        submitTransactionText
    };
}
