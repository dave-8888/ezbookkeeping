<template>
    <v-row>
        <v-col cols="12">
            <v-card :title="tt('AI Settings')" :class="{ disabled: loading }">
                <v-form @submit.prevent="save">
                    <v-card-text>
                        <v-row>
                            <v-col cols="12" md="6">
                                <v-select
                                    item-title="displayName"
                                    item-value="value"
                                    persistent-placeholder
                                    :label="tt('Enabled')"
                                    :placeholder="tt('Enabled')"
                                    :items="enableDisableOptions"
                                    v-model="form.enabled"
                                />
                            </v-col>

                            <v-col cols="12" md="6">
                                <v-select
                                    item-title="name"
                                    item-value="value"
                                    persistent-placeholder
                                    :disabled="!form.enabled"
                                    :label="tt('Large Language Model Provider')"
                                    :placeholder="tt('Large Language Model Provider')"
                                    :items="allLLMProviders"
                                    v-model="form.llmProvider"
                                />
                            </v-col>

                            <template v-if="form.enabled && form.llmProvider === LLMProvider.OpenAI">
                                <v-col cols="12" md="6">
                                    <v-text-field
                                        type="password"
                                        autocomplete="off"
                                        persistent-placeholder
                                        :label="secretLabel('OpenAI API Key', llmSetting?.openAIAPIKeySet)"
                                        :placeholder="secretPlaceholder(llmSetting?.openAIAPIKeySet)"
                                        v-model="form.openAIAPIKey"
                                    />
                                </v-col>

                                <v-col cols="12" md="6">
                                    <v-text-field
                                        persistent-placeholder
                                        :label="tt('OpenAI Model ID')"
                                        :placeholder="tt('OpenAI Model ID')"
                                        v-model="form.openAIModelID"
                                    />
                                </v-col>
                            </template>

                            <template v-if="form.enabled && form.llmProvider === LLMProvider.OpenAICompatible">
                                <v-col cols="12" md="6">
                                    <v-text-field
                                        persistent-placeholder
                                        :label="tt('OpenAI Compatible Base URL')"
                                        :placeholder="tt('OpenAI Compatible Base URL')"
                                        v-model="form.openAICompatibleBaseURL"
                                    />
                                </v-col>

                                <v-col cols="12" md="6">
                                    <v-text-field
                                        type="password"
                                        autocomplete="off"
                                        persistent-placeholder
                                        :label="secretLabel('OpenAI Compatible API Key', llmSetting?.openAICompatibleAPIKeySet)"
                                        :placeholder="secretPlaceholder(llmSetting?.openAICompatibleAPIKeySet)"
                                        v-model="form.openAICompatibleAPIKey"
                                    />
                                </v-col>

                                <v-col cols="12" md="6">
                                    <v-text-field
                                        persistent-placeholder
                                        :label="tt('OpenAI Compatible Model ID')"
                                        :placeholder="tt('OpenAI Compatible Model ID')"
                                        v-model="form.openAICompatibleModelID"
                                    />
                                </v-col>
                            </template>

                            <template v-if="form.enabled && form.llmProvider === LLMProvider.OpenRouter">
                                <v-col cols="12" md="6">
                                    <v-text-field
                                        type="password"
                                        autocomplete="off"
                                        persistent-placeholder
                                        :label="secretLabel('OpenRouter API Key', llmSetting?.openRouterAPIKeySet)"
                                        :placeholder="secretPlaceholder(llmSetting?.openRouterAPIKeySet)"
                                        v-model="form.openRouterAPIKey"
                                    />
                                </v-col>

                                <v-col cols="12" md="6">
                                    <v-text-field
                                        persistent-placeholder
                                        :label="tt('OpenRouter Model ID')"
                                        :placeholder="tt('OpenRouter Model ID')"
                                        v-model="form.openRouterModelID"
                                    />
                                </v-col>
                            </template>

                            <template v-if="form.enabled && form.llmProvider === LLMProvider.Ollama">
                                <v-col cols="12" md="6">
                                    <v-text-field
                                        persistent-placeholder
                                        :label="tt('Ollama Server URL')"
                                        :placeholder="tt('Ollama Server URL')"
                                        v-model="form.ollamaServerURL"
                                    />
                                </v-col>

                                <v-col cols="12" md="6">
                                    <v-text-field
                                        persistent-placeholder
                                        :label="tt('Ollama Model ID')"
                                        :placeholder="tt('Ollama Model ID')"
                                        v-model="form.ollamaModelID"
                                    />
                                </v-col>
                            </template>

                            <template v-if="form.enabled && form.llmProvider === LLMProvider.GoogleAI">
                                <v-col cols="12" md="6">
                                    <v-text-field
                                        type="password"
                                        autocomplete="off"
                                        persistent-placeholder
                                        :label="secretLabel('Google AI API Key', llmSetting?.googleAIAPIKeySet)"
                                        :placeholder="secretPlaceholder(llmSetting?.googleAIAPIKeySet)"
                                        v-model="form.googleAIAPIKey"
                                    />
                                </v-col>

                                <v-col cols="12" md="6">
                                    <v-text-field
                                        persistent-placeholder
                                        :label="tt('Google AI Model ID')"
                                        :placeholder="tt('Google AI Model ID')"
                                        v-model="form.googleAIModelID"
                                    />
                                </v-col>
                            </template>

                            <v-col cols="12" md="4">
                                <v-text-field
                                    type="number"
                                    min="0"
                                    persistent-placeholder
                                    :disabled="!form.enabled"
                                    :label="tt('Request Timeout')"
                                    :placeholder="tt('Request Timeout')"
                                    v-model.number="form.largeLanguageModelAPIRequestTimeout"
                                />
                            </v-col>

                            <v-col cols="12" md="4">
                                <v-text-field
                                    persistent-placeholder
                                    :disabled="!form.enabled"
                                    :label="tt('Proxy')"
                                    :placeholder="tt('Proxy')"
                                    v-model="form.largeLanguageModelAPIProxy"
                                />
                            </v-col>

                            <v-col cols="12" md="4">
                                <v-select
                                    item-title="displayName"
                                    item-value="value"
                                    persistent-placeholder
                                    :disabled="!form.enabled"
                                    :label="tt('Skip TLS Verification')"
                                    :placeholder="tt('Skip TLS Verification')"
                                    :items="enableDisableOptions"
                                    v-model="form.largeLanguageModelAPISkipTLSVerify"
                                />
                            </v-col>
                        </v-row>
                    </v-card-text>

                    <v-card-actions>
                        <v-btn color="primary" type="submit" :loading="saving" :disabled="loading || saving">
                            {{ tt('Save') }}
                        </v-btn>
                    </v-card-actions>
                </v-form>
            </v-card>
        </v-col>

        <snack-bar ref="snackbar" />
    </v-row>
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, reactive, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserLLMSettingsStore } from '@/stores/userLLMSetting.ts';

import type { LocalizedSwitchOption, NameValue } from '@/core/base.ts';
import type { UserLLMSettingResponse, UserLLMSettingUpdateRequest } from '@/models/user_llm_setting.ts';

enum LLMProvider {
    OpenAI = 'openai',
    OpenAICompatible = 'openai_compatible',
    OpenRouter = 'openrouter',
    Ollama = 'ollama',
    GoogleAI = 'google_ai'
}

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt, getAllEnableDisableOptions } = useI18n();

const userLLMSettingsStore = useUserLLMSettingsStore();

const snackbar = useTemplateRef<SnackBarType>('snackbar');

const loading = ref<boolean>(false);
const saving = ref<boolean>(false);
const llmSetting = ref<UserLLMSettingResponse | null>(null);

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

const enableDisableOptions = computed<LocalizedSwitchOption[]>(() => getAllEnableDisableOptions());
const allLLMProviders = computed<NameValue[]>(() => [
    { name: tt('OpenAI'), value: LLMProvider.OpenAI },
    { name: tt('OpenAI Compatible'), value: LLMProvider.OpenAICompatible },
    { name: tt('OpenRouter'), value: LLMProvider.OpenRouter },
    { name: tt('Ollama'), value: LLMProvider.Ollama },
    { name: tt('Google AI'), value: LLMProvider.GoogleAI }
]);

function secretLabel(label: string, configured?: boolean): string {
    if (configured) {
        return `${tt(label)} (${tt('Configured')})`;
    }

    return tt(label);
}

function secretPlaceholder(configured?: boolean): string {
    if (configured) {
        return tt('Leave blank to keep existing value');
    }

    return '';
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
            snackbar.value?.showError(error);
        }
    }).finally(() => {
        loading.value = false;
    });
}

function save(): void {
    saving.value = true;

    userLLMSettingsStore.updateUserLLMSetting(form).then(setting => {
        fillForm(setting);
        snackbar.value?.showMessage('AI settings have been saved');
    }).catch(error => {
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    }).finally(() => {
        saving.value = false;
    });
}

load();
</script>
