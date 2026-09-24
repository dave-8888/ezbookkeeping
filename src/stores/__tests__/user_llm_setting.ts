import { beforeEach, describe, expect, jest, test } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

import services from '@/lib/services.ts';
import { isServerTransactionAIConfigured } from '@/lib/server_settings.ts';
import { useUserLLMSettingsStore } from '@/stores/userLLMSetting.ts';

jest.mock('@/lib/services.ts', () => ({
    __esModule: true,
    default: { getUserLLMSetting: jest.fn(), updateUserLLMSetting: jest.fn() }
}));
jest.mock('@/lib/logger.ts', () => ({ __esModule: true, default: { error: jest.fn() } }));
jest.mock('@/lib/server_settings.ts', () => ({ isServerTransactionAIConfigured: jest.fn() }));

describe('transaction AI availability', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
        jest.mocked(isServerTransactionAIConfigured).mockReturnValue(false);
    });

    test('stays disabled before settings load and when no AI is configured', async () => {
        const store = useUserLLMSettingsStore();
        expect(store.transactionAIEnabled).toBe(false);
        jest.mocked(services.getUserLLMSetting).mockResolvedValue({
            data: { success: true, result: { enabled: false, llmProvider: '' } }
        } as never);
        await store.loadUserLLMSetting({ force: false });
        expect(store.transactionAIEnabled).toBe(false);
    });

    test('reacts to personal AI being enabled, disabled and reset', async () => {
        const store = useUserLLMSettingsStore();
        jest.mocked(services.getUserLLMSetting).mockResolvedValue({
            data: { success: true, result: { enabled: true, llmProvider: 'openai' } }
        } as never);
        await store.loadUserLLMSetting({ force: false });
        expect(store.transactionAIEnabled).toBe(true);
        jest.mocked(services.updateUserLLMSetting).mockResolvedValue({
            data: { success: true, result: { enabled: false, llmProvider: 'openai' } }
        } as never);
        await store.updateUserLLMSetting({ enabled: false } as never);
        expect(store.transactionAIEnabled).toBe(false);
        store.resetUserLLMSetting();
        expect(store.transactionAIEnabled).toBe(false);
    });

    test('allows configured server AI without personal AI settings', () => {
        jest.mocked(isServerTransactionAIConfigured).mockReturnValue(true);
        expect(useUserLLMSettingsStore().transactionAIEnabled).toBe(true);
    });

    test('keeps AI disabled if loading personal settings fails', async () => {
        const store = useUserLLMSettingsStore();
        jest.mocked(services.getUserLLMSetting).mockRejectedValue(new Error('offline'));
        await expect(store.loadUserLLMSetting({ force: false })).rejects.toBeDefined();
        expect(store.transactionAIEnabled).toBe(false);
    });
});
