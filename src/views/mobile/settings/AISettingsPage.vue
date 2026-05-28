<template>
    <f7-page>
        <f7-navbar :title="tt('AI Settings')" :back-link="tt('Back')"></f7-navbar>

        <f7-list strong inset dividers class="settings-list" :class="{ 'disabled': loading }">
            <f7-list-item>
                <template #after-title>
                    {{ tt('Enabled') }}
                </template>
                <template #after>
                    <f7-toggle :checked="form.enabled" @toggle:change="form.enabled = $event"></f7-toggle>
                </template>
            </f7-list-item>

            <f7-list-item link="#" :title="tt('Large Language Model Provider')" :after="findNameByValue(allLLMProviders, form.llmProvider)" :disabled="!form.enabled" @click="showProviderPopup = true">
                <list-item-selection-popup value-type="item"
                                           key-field="value" value-field="value"
                                           title-field="name"
                                           :title="tt('Large Language Model Provider')"
                                           :enable-filter="true"
                                           :filter-placeholder="tt('Large Language Model Provider')"
                                           :filter-no-items-text="tt('No results')"
                                           :items="allLLMProviders"
                                           v-model:show="showProviderPopup"
                                           v-model="form.llmProvider">
                </list-item-selection-popup>
            </f7-list-item>
        </f7-list>

        <f7-list strong inset form dividers v-if="form.enabled">
            <template v-if="form.llmProvider === LLMProvider.OpenAI">
                <f7-list-input type="password" autocomplete="off"
                               :label="secretLabel('OpenAI API Key', llmSetting?.openAIAPIKeySet)"
                               :placeholder="secretPlaceholder(llmSetting?.openAIAPIKeySet)"
                               v-model:value="form.openAIAPIKey"></f7-list-input>
                <f7-list-input type="text" :label="tt('OpenAI Model ID')" :placeholder="tt('OpenAI Model ID')" v-model:value="form.openAIModelID"></f7-list-input>
            </template>

            <template v-if="form.llmProvider === LLMProvider.OpenAICompatible">
                <f7-list-input type="text" :label="tt('OpenAI Compatible Base URL')" :placeholder="tt('OpenAI Compatible Base URL')" v-model:value="form.openAICompatibleBaseURL"></f7-list-input>
                <f7-list-input type="password" autocomplete="off"
                               :label="secretLabel('OpenAI Compatible API Key', llmSetting?.openAICompatibleAPIKeySet)"
                               :placeholder="secretPlaceholder(llmSetting?.openAICompatibleAPIKeySet)"
                               v-model:value="form.openAICompatibleAPIKey"></f7-list-input>
                <f7-list-input type="text" :label="tt('OpenAI Compatible Model ID')" :placeholder="tt('OpenAI Compatible Model ID')" v-model:value="form.openAICompatibleModelID"></f7-list-input>
            </template>

            <template v-if="form.llmProvider === LLMProvider.OpenRouter">
                <f7-list-input type="password" autocomplete="off"
                               :label="secretLabel('OpenRouter API Key', llmSetting?.openRouterAPIKeySet)"
                               :placeholder="secretPlaceholder(llmSetting?.openRouterAPIKeySet)"
                               v-model:value="form.openRouterAPIKey"></f7-list-input>
                <f7-list-input type="text" :label="tt('OpenRouter Model ID')" :placeholder="tt('OpenRouter Model ID')" v-model:value="form.openRouterModelID"></f7-list-input>
            </template>

            <template v-if="form.llmProvider === LLMProvider.Ollama">
                <f7-list-input type="text" :label="tt('Ollama Server URL')" :placeholder="tt('Ollama Server URL')" v-model:value="form.ollamaServerURL"></f7-list-input>
                <f7-list-input type="text" :label="tt('Ollama Model ID')" :placeholder="tt('Ollama Model ID')" v-model:value="form.ollamaModelID"></f7-list-input>
            </template>

            <template v-if="form.llmProvider === LLMProvider.GoogleAI">
                <f7-list-input type="password" autocomplete="off"
                               :label="secretLabel('Google AI API Key', llmSetting?.googleAIAPIKeySet)"
                               :placeholder="secretPlaceholder(llmSetting?.googleAIAPIKeySet)"
                               v-model:value="form.googleAIAPIKey"></f7-list-input>
                <f7-list-input type="text" :label="tt('Google AI Model ID')" :placeholder="tt('Google AI Model ID')" v-model:value="form.googleAIModelID"></f7-list-input>
            </template>

            <f7-list-input type="number" min="0" :label="tt('Request Timeout')" :placeholder="tt('Request Timeout')" v-model:value="form.largeLanguageModelAPIRequestTimeout"></f7-list-input>
            <f7-list-input type="text" :label="tt('Proxy')" :placeholder="tt('Proxy')" v-model:value="form.largeLanguageModelAPIProxy"></f7-list-input>
            <f7-list-item>
                <template #after-title>
                    {{ tt('Skip TLS Verification') }}
                </template>
                <template #after>
                    <f7-toggle :checked="form.largeLanguageModelAPISkipTLSVerify" @toggle:change="form.largeLanguageModelAPISkipTLSVerify = $event"></f7-toggle>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-block>
            <f7-button fill :class="{ disabled: loading || saving }" @click="save">
                <f7-preloader color="white" :size="16" v-if="saving" />
                <span v-else>{{ tt('Save') }}</span>
            </f7-button>
        </f7-block>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useUserLLMSettingsStore } from '@/stores/userLLMSetting.ts';

import type { NameValue } from '@/core/base.ts';
import type { UserLLMSettingResponse, UserLLMSettingUpdateRequest } from '@/models/user_llm_setting.ts';

import { findNameByValue } from '@/lib/common.ts';

enum LLMProvider {
    OpenAI = 'openai',
    OpenAICompatible = 'openai_compatible',
    OpenRouter = 'openrouter',
    Ollama = 'ollama',
    GoogleAI = 'google_ai'
}

const { tt } = useI18n();
const { showToast } = useI18nUIComponents();

const userLLMSettingsStore = useUserLLMSettingsStore();

const loading = ref<boolean>(false);
const saving = ref<boolean>(false);
const showProviderPopup = ref<boolean>(false);
const llmSetting = ref<UserLLMSettingResponse | null>(null);

const allLLMProviders = computed<NameValue[]>(() => [
    { name: tt('OpenAI'), value: LLMProvider.OpenAI },
    { name: tt('OpenAI Compatible'), value: LLMProvider.OpenAICompatible },
    { name: tt('OpenRouter'), value: LLMProvider.OpenRouter },
    { name: tt('Ollama'), value: LLMProvider.Ollama },
    { name: tt('Google AI'), value: LLMProvider.GoogleAI }
]);

const form = reactive<UserLLMSettingUpdateRequest>({
    enabled: false,
    llmProvider: LLMProvider.OpenAI,
    openAIAPIKey: '',
    openAIModelID: '',
    openAICompatibleBaseURL: '',
    openAICompatibleAPIKey: '',
    openAICompatibleModelID: '',
    openRouterAPIKey: '',
    openRouterModelID: '',
    ollamaServerURL: '',
    ollamaModelID: '',
    googleAIAPIKey: '',
    googleAIModelID: '',
    largeLanguageModelAPIRequestTimeout: 60000,
    largeLanguageModelAPIProxy: 'system',
    largeLanguageModelAPISkipTLSVerify: false
});

function secretLabel(label: string, configured?: boolean): string {
    if (configured) {
        return `${tt(label)} (${tt('Configured')})`;
    }

    return tt(label);
}

function secretPlaceholder(configured?: boolean): string {
    return configured ? tt('Leave blank to keep existing value') : '';
}

function fillForm(setting: UserLLMSettingResponse): void {
    llmSetting.value = setting;
    form.enabled = setting.enabled;
    form.llmProvider = setting.llmProvider || LLMProvider.OpenAI;
    form.openAIAPIKey = '';
    form.openAIModelID = setting.openAIModelID || '';
    form.openAICompatibleBaseURL = setting.openAICompatibleBaseURL || '';
    form.openAICompatibleAPIKey = '';
    form.openAICompatibleModelID = setting.openAICompatibleModelID || '';
    form.openRouterAPIKey = '';
    form.openRouterModelID = setting.openRouterModelID || '';
    form.ollamaServerURL = setting.ollamaServerURL || '';
    form.ollamaModelID = setting.ollamaModelID || '';
    form.googleAIAPIKey = '';
    form.googleAIModelID = setting.googleAIModelID || '';
    form.largeLanguageModelAPIRequestTimeout = setting.largeLanguageModelAPIRequestTimeout || 60000;
    form.largeLanguageModelAPIProxy = setting.largeLanguageModelAPIProxy || 'system';
    form.largeLanguageModelAPISkipTLSVerify = setting.largeLanguageModelAPISkipTLSVerify;
}

function load(): void {
    loading.value = true;

    userLLMSettingsStore.loadUserLLMSetting({ force: false }).then(setting => {
        fillForm(setting);
    }).catch(error => {
        if (!error.processed) {
            showToast(error.message || error);
        }
    }).finally(() => {
        loading.value = false;
    });
}

function save(): void {
    saving.value = true;
    showLoading(() => saving.value);

    userLLMSettingsStore.updateUserLLMSetting({
        ...form,
        largeLanguageModelAPIRequestTimeout: Number(form.largeLanguageModelAPIRequestTimeout) || 0
    }).then(setting => {
        fillForm(setting);
        showToast('AI settings have been saved');
    }).catch(error => {
        if (!error.processed) {
            showToast(error.message || error);
        }
    }).finally(() => {
        saving.value = false;
        hideLoading();
    });
}

load();
</script>
