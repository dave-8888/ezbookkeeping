import { beforeEach, describe, expect, jest, test } from '@jest/globals';
import { effectScope } from 'vue';
import { useQuickTransactionTextInput } from '@/components/base/QuickTransactionTextInputBase.ts';
import { useTransactionsStore } from '@/stores/transaction.ts';

jest.mock('@/stores/transaction.ts', () => ({ useTransactionsStore: jest.fn() }));
jest.mock('@/stores/user.ts', () => ({ useUserStore: () => ({ currentUserBasicInfo: { username: 'test-user' } }) }));
jest.mock('@/stores/userLLMSetting.ts', () => ({ useUserLLMSettingsStore: () => ({ transactionAIEnabled: true, loadUserLLMSetting: async () => undefined }) }));
jest.mock('@/lib/misc.ts', () => {
    let sequence = 0;
    return { generateRandomUUID: () => `request-${++sequence}` };
});

function setup() {
    const create = jest.fn<ReturnType<typeof useTransactionsStore>['createTransactionByMCP']>();
    jest.mocked(useTransactionsStore).mockReturnValue({ createTransactionByMCP: create } as never);
    const onSaved = jest.fn();
    const onNeedManualInput = jest.fn<() => void>();
    const scope = effectScope();
    const state = scope.run(() => useQuickTransactionTextInput({ showMessage: jest.fn(), showError: jest.fn(), onSaved, onNeedManualInput }))!;
    state.transactionText.value = '昨天现金午饭20元';
    return { state, create, scope, onSaved, onNeedManualInput };
}

beforeEach(() => {
    const values = new Map<string, string>();
    Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, value: {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => values.set(key, value),
        removeItem: (key: string) => values.delete(key)
    } });
});

describe('MCP quick transaction submission', () => {
    test('uses the MCP write result, clears input and refreshes after success', async () => {
        const { state, create, scope, onSaved, onNeedManualInput } = setup();
        create.mockResolvedValue({ status: 'created', reply: '已记账' });
        await state.submitTransactionText();
        expect(create).toHaveBeenCalledTimes(1);
        expect(onSaved).toHaveBeenCalledTimes(1);
        expect(onNeedManualInput).not.toHaveBeenCalled();
        expect(state.transactionText.value).toBe('');
        expect(state.recognizingByText.value).toBe(false);
        scope.stop();
    });

    test('retains request identity after network failure and remount, including other submissions', async () => {
        const first = setup();
        first.create.mockRejectedValue(new Error('network timeout'));
        await first.state.submitTransactionText();
        const request = first.create.mock.calls[0]![0];
        first.state.transactionText.value = '今天现金车费10元';
        await first.state.submitTransactionText();
        first.scope.stop();
        const retry = setup();
        retry.create.mockResolvedValue({ status: 'processing', reply: '处理中' });
        await retry.state.submitTransactionText();
        expect(retry.create.mock.calls[0]![0]).toEqual(request);
        expect(retry.onSaved).not.toHaveBeenCalled();
        expect(retry.state.transactionText.value).toBe(request.text);
        retry.scope.stop();
    });

    test('suppresses double clicks and retains unknown requests', async () => {
        const { state, create, scope, onSaved } = setup();
        create.mockResolvedValue({ status: 'unknown', reply: '请核对' });
        await Promise.all([state.submitTransactionText(), state.submitTransactionText()]);
        expect(create).toHaveBeenCalledTimes(1);
        await state.submitTransactionText();
        expect(create.mock.calls[0]![0]).toEqual(create.mock.calls[1]![0]);
        expect(onSaved).not.toHaveBeenCalled();
        scope.stop();
    });
});
