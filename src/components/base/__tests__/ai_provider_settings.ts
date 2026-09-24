import { describe, expect, jest, test } from '@jest/globals';
import { effectScope, reactive, ref } from 'vue';

import services from '@/lib/services.ts';
import { useAIProviderSettings } from '@/components/base/AIProviderSettingsBase.ts';
import type { UserLLMSettingUpdateRequest } from '@/models/user_llm_setting.ts';

jest.mock('@/lib/services.ts', () => ({ __esModule: true, default: { getLLMProviders: jest.fn(), getLLMModels: jest.fn() } }));

function setup() {
    const form = reactive({ enabled: true, llmProvider: 'openai', openAICompatibleBaseURL: '', openAICompatibleAPIKey: '', openAICompatibleModelID: '', openAIModelID: 'existing', largeLanguageModelAPIRequestTimeout: 60000 } as UserLLMSettingUpdateRequest);
    const scope = effectScope();
    const state = scope.run(() => useAIProviderSettings(form, ref(null)))!;
    jest.mocked(services.getLLMProviders).mockResolvedValue({ data: { success: true, result: [
        { id: 'openai_compatible', name: 'Custom', protocol: 'openai_compatible', baseUrl: '' },
        { id: 'domestic', name: 'Domestic', protocol: 'openai_compatible', baseUrl: 'https://domestic.example/v1' },
        { id: 'other', name: 'Other', protocol: 'openai_compatible', baseUrl: 'https://other.example/v1' }
    ] } } as never);
    return { form, scope, state };
}

describe('AI settings provider and model interactions', () => {
    test('fills presets, isolates keys and permits switching to custom', async () => {
        const { form, scope, state } = setup();
        await state.refreshProviders();
        state.selectedProvider.value = 'domestic';
        expect(form.llmProvider).toBe('openai_compatible');
        expect(form.openAICompatibleBaseURL).toBe('https://domestic.example/v1');
        form.openAICompatibleAPIKey = 'typed-key';
        state.modelId.value = 'manual-model';
        state.selectedProvider.value = 'other';
        expect(form.openAICompatibleAPIKey).toBe('');
        expect(state.modelId.value).toBe('');
        state.selectedProvider.value = 'openai_compatible';
        expect(state.selectedProvider.value).toBe('openai_compatible');
        scope.stop();
    });

    test('identifies saved presets regardless of catalog ordering', async () => {
        const { form, scope, state } = setup();
        form.llmProvider = 'openai_compatible';
        form.openAICompatibleBaseURL = 'https://domestic.example/v1/';
        await state.refreshProviders();
        expect(state.selectedProvider.value).toBe('domestic');
        scope.stop();
    });

    test('refreshes models without replacing a manually selected model', async () => {
        const { scope, state } = setup();
        jest.mocked(services.getLLMModels).mockResolvedValue({ data: { success: true, result: ['new-model'] } } as never);
        await state.refreshModels();
        expect(state.models.value).toEqual(['new-model']);
        expect(state.modelId.value).toBe('existing');
        jest.mocked(services.getLLMModels).mockRejectedValue(new Error('offline'));
        await state.refreshModels();
        expect(state.modelStatus.value).toContain('Unable to refresh');
        expect(state.modelId.value).toBe('existing');
        scope.stop();
    });

    test('ignores an old response after switching providers', async () => {
        const { form, scope, state } = setup();
        let finish!: (value: never) => void;
        jest.mocked(services.getLLMModels).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
        const pending = state.refreshModels();
        form.llmProvider = 'ollama';
        finish({ data: { success: true, result: ['old-model'] } } as never);
        await pending;
        expect(state.models.value).toEqual([]);
        expect(state.loadingModels.value).toBe(false);
        scope.stop();
    });

    test('applies refreshed provider names while preserving form data on failure', async () => {
        const { form, scope, state } = setup();
        await state.refreshProviders();
        state.selectedProvider.value = 'domestic';
        jest.mocked(services.getLLMProviders).mockResolvedValue({ data: { success: true, result: [
            { id: 'domestic', name: 'Renamed', protocol: 'openai_compatible', baseUrl: form.openAICompatibleBaseURL }
        ] } } as never);
        await state.refreshProviders();
        expect(state.selectedPreset.value?.name).toBe('Renamed');
        jest.mocked(services.getLLMProviders).mockRejectedValue(new Error('offline'));
        await state.refreshProviders();
        expect(state.selectedPreset.value?.name).toBe('Renamed');
        expect(state.providerError.value).toBeTruthy();
        scope.stop();
    });
});
