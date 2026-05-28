import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import type {
    UserLLMSettingResponse,
    UserLLMSettingUpdateRequest
} from '@/models/user_llm_setting.ts';

import services from '@/lib/services.ts';
import logger from '@/lib/logger.ts';

export const useUserLLMSettingsStore = defineStore('userLLMSettings', () => {
    const userLLMSetting = ref<UserLLMSettingResponse | null>(null);
    const userLLMSettingStateInvalid = ref<boolean>(true);

    const enabled = computed<boolean>(() => !!userLLMSetting.value?.enabled && !!userLLMSetting.value?.llmProvider);

    function loadUserLLMSetting({ force }: { force: boolean }): Promise<UserLLMSettingResponse> {
        return new Promise((resolve, reject) => {
            if (!force && !userLLMSettingStateInvalid.value && userLLMSetting.value) {
                resolve(userLLMSetting.value);
                return;
            }

            services.getUserLLMSetting().then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve AI settings' });
                    return;
                }

                userLLMSetting.value = data.result;
                userLLMSettingStateInvalid.value = false;
                resolve(data.result);
            }).catch(error => {
                logger.error('failed to load user llm setting', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to retrieve AI settings' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function updateUserLLMSetting(req: UserLLMSettingUpdateRequest): Promise<UserLLMSettingResponse> {
        return new Promise((resolve, reject) => {
            services.updateUserLLMSetting(req).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to save AI settings' });
                    return;
                }

                userLLMSetting.value = data.result;
                userLLMSettingStateInvalid.value = false;
                resolve(data.result);
            }).catch(error => {
                logger.error('failed to update user llm setting', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to save AI settings' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function resetUserLLMSetting(): void {
        userLLMSetting.value = null;
        userLLMSettingStateInvalid.value = true;
    }

    return {
        userLLMSetting,
        userLLMSettingStateInvalid,
        enabled,
        loadUserLLMSetting,
        updateUserLLMSetting,
        resetUserLLMSetting
    };
});
