import { computed, ref, watch, type Ref } from 'vue';

import type { LLMProviderPreset, UserLLMSettingResponse, UserLLMSettingUpdateRequest } from '@/models/user_llm_setting.ts';
import services from '@/lib/services.ts';

export function sameAIEndpoint(a: string, b: string): boolean {
    return a.trim().replace(/\/+$/, '') === b.trim().replace(/\/+$/, '');
}

export function findAIProvider(providers: LLMProviderPreset[], form: UserLLMSettingUpdateRequest): LLMProviderPreset | undefined {
    return providers.find(provider => provider.protocol === form.llmProvider && provider.baseUrl &&
        sameAIEndpoint(provider.baseUrl, form.openAICompatibleBaseURL)) ||
        providers.find(provider => provider.protocol === form.llmProvider && !provider.baseUrl && provider.id === form.llmProvider);
}

export function useAIProviderSettings(form: UserLLMSettingUpdateRequest, saved: Ref<UserLLMSettingResponse | null>) {
    const providers = ref<LLMProviderPreset[]>([]);
    const loadingProviders = ref(false);
    const loadingModels = ref(false);
    const models = ref<string[]>([]);
    const providerError = ref('');
    const modelStatus = ref('');
    let modelRequestVersion = 0;

    const explicitProvider = ref('');
    const selectedPreset = computed(() => {
        const explicit = providers.value.find(provider => provider.id === explicitProvider.value && provider.protocol === form.llmProvider &&
            (!provider.baseUrl || sameAIEndpoint(provider.baseUrl, form.openAICompatibleBaseURL)));
        return explicit || findAIProvider(providers.value, form);
    });
    const selectedProvider = computed<string>({
        get: () => selectedPreset.value?.id || form.llmProvider,
        set: id => {
            const provider = providers.value.find(item => item.id === id);
            if (!provider) { return; }
            if (provider.baseUrl && !sameAIEndpoint(provider.baseUrl, form.openAICompatibleBaseURL)) {
                form.openAICompatibleAPIKey = '';
                form.openAICompatibleModelID = '';
                form.openAICompatibleBaseURL = provider.baseUrl;
            }
            form.llmProvider = provider.protocol;
            explicitProvider.value = id;
        }
    });
    const providerOptions = computed(() => {
        const options = providers.value.map(provider => ({ name: provider.name, value: provider.id }));
        if (!options.some(option => option.value === selectedProvider.value)) {
            options.push({ name: form.llmProvider, value: form.llmProvider });
        }
        return options;
    });
    const modelField = computed<keyof UserLLMSettingUpdateRequest>(() => {
        switch (form.llmProvider) {
            case 'openai': return 'openAIModelID';
            case 'openrouter': return 'openRouterModelID';
            case 'ollama': return 'ollamaModelID';
            case 'google_ai': return 'googleAIModelID';
            default: return 'openAICompatibleModelID';
        }
    });
    const modelId = computed<string>({
        get: () => String(form[modelField.value] || ''),
        set: value => { Object.assign(form, { [modelField.value]: value || '' }); }
    });
    const compatibleKeyConfigured = computed(() => !!saved.value?.openAICompatibleAPIKeySet &&
        sameAIEndpoint(form.openAICompatibleBaseURL, saved.value.openAICompatibleBaseURL));

    watch(() => [form.llmProvider, form.openAICompatibleBaseURL, form.ollamaServerURL,
        form.openAIAPIKey, form.openAICompatibleAPIKey, form.openRouterAPIKey, form.googleAIAPIKey, form.enabled], () => {
        modelRequestVersion++;
        models.value = [];
        modelStatus.value = '';
        loadingModels.value = false;
    }, { flush: 'sync' });

    async function refreshProviders(): Promise<void> {
        if (loadingProviders.value) { return; }
        loadingProviders.value = true;
        providerError.value = '';
        try {
            const response = await services.getLLMProviders();
            if (!response.data?.success || !Array.isArray(response.data.result)) { throw new Error(); }
            providers.value = response.data.result;
        } catch {
            providerError.value = 'Unable to refresh providers';
        } finally {
            loadingProviders.value = false;
        }
    }

    async function refreshModels(): Promise<void> {
        if (loadingModels.value || !form.enabled) { return; }
        const version = ++modelRequestVersion;
        loadingModels.value = true;
        modelStatus.value = '';
        try {
            const response = await services.getLLMModels({ ...form,
                largeLanguageModelAPIRequestTimeout: Number(form.largeLanguageModelAPIRequestTimeout) || 0 });
            if (version !== modelRequestVersion) { return; }
            if (!response.data?.success || !Array.isArray(response.data.result)) { throw new Error(); }
            models.value = response.data.result;
            modelStatus.value = models.value.length ? 'Model list refreshed' : 'No models returned. Enter a model ID manually.';
        } catch {
            if (version === modelRequestVersion) {
                modelStatus.value = 'Unable to refresh models. Check the endpoint and API key, or enter a model ID manually.';
            }
        } finally {
            if (version === modelRequestVersion) { loadingModels.value = false; }
        }
    }

    return { selectedProvider, selectedPreset, providerOptions, modelId, models,
        compatibleKeyConfigured, loadingProviders, loadingModels, providerError, modelStatus,
        refreshProviders, refreshModels };
}
