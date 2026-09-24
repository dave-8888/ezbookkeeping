import { computed, onScopeDispose, ref, watch } from 'vue';

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
    continuous?: boolean;
    formatError?: (error: unknown) => string;
}

export interface QuickTransactionSubmission {
    id: string;
    text: string;
    status: 'pending' | 'created' | 'attention';
    message: string;
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
    const activeRequests = ref(0);
    const recognizingByText = computed(() => activeRequests.value > 0);
    const submissions = ref<QuickTransactionSubmission[]>([]);
    const timers = new Set<ReturnType<typeof setTimeout>>();
    let disposed = false;

    function dismissSubmission(id: string): void {
        submissions.value = submissions.value.filter(item => item.id !== id);
    }

    onScopeDispose(() => {
        disposed = true;
        timers.forEach(timer => clearTimeout(timer));
    });

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
        if (!options.continuous && recognizingByText.value) {
            return Promise.resolve();
        }

        if (!aiEnabled.value) {
            options.showError('llm provider is not enabled');
            return Promise.resolve();
        }

        const text = (transactionText.value || '').trim();

        if (!text) {
            options.showError('Parameter Invalid');
            return Promise.resolve();
        }

        // Identical in-flight text shares an idempotency key; do not display or
        // execute it twice. Different entries may finish in any order.
        if (options.continuous && submissions.value.some(item => item.text === text && item.status === 'pending')) {
            return Promise.resolve();
        }

        const submission = ref<QuickTransactionSubmission>({
            id: generateRandomUUID(), text, status: 'pending', message: ''
        });
        if (options.continuous) {
            submissions.value.push(submission.value);
            clearTransactionText();
        }
        activeRequests.value++;

        return Promise.resolve().then(() => {
            return transactionsStore.createTransactionByMCP(pendingRequest(text));
        }).then(result => {
            if (result.status === 'created') {
                forgetPendingRequest(text);
                if (options.continuous) {
                    submission.value.status = 'created';
                    submission.value.message = result.reply;
                    if (!disposed) {
                        const timer = setTimeout(() => {
                            dismissSubmission(submission.value.id);
                            timers.delete(timer);
                        }, 3000);
                        timers.add(timer);
                    }
                } else {
                    clearTransactionText();
                    options.showMessage(result.reply);
                }
                options.onSaved?.();
            } else {
                if (result.status === 'failed' || result.status === 'needs_input') {
                    forgetPendingRequest(text);
                }

                if (options.continuous) {
                    submission.value.status = 'attention';
                    submission.value.message = result.reply;
                } else {
                    options.showMessage(result.reply);
                }
            }
        }).catch(error => {
            if (options.continuous) {
                submission.value.status = 'attention';
                submission.value.message = options.formatError?.(error) || 'operation failed';
            } else {
                options.showError(error);
            }
        }).finally(() => {
            activeRequests.value--;
        });
    }

    return {
        submissions,
        dismissSubmission,
        aiEnabled,
        transactionText,
        recognizingByText,
        clearTransactionText,
        submitTransactionText
    };
}
