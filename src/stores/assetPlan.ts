import { ref } from 'vue';
import { defineStore } from 'pinia';

import type {
    AssetPlanCreateRequest,
    AssetPlanModifyRequest,
    AssetPlanInfoResponse
} from '@/models/asset_plan.ts';

import logger from '@/lib/logger.ts';
import services from '@/lib/services.ts';

function getRequestError(error: Record<string, unknown>, fallbackMessage: string): Record<string, unknown> {
    const response = error?.['response'] as { data?: { errorMessage?: string } } | undefined;

    if (response?.data?.errorMessage) {
        return { error: response.data, message: response.data.errorMessage };
    } else if (!error?.['processed']) {
        return { message: fallbackMessage };
    }

    return error;
}

export const useAssetPlanStore = defineStore('assetPlan', () => {
    const assetPlan = ref<AssetPlanInfoResponse | null>(null);

    function loadAssetPlan(detailMonth?: string): Promise<AssetPlanInfoResponse | null> {
        return services.getAssetPlan(detailMonth).then(response => {
            const data = response.data;

            if (!data?.success) {
                return Promise.reject({ message: 'Unable to retrieve asset plan' });
            }

            assetPlan.value = data.result ?? null;
            return assetPlan.value;
        }).catch(error => {
            logger.error('failed to retrieve asset plan', error);
            return Promise.reject(getRequestError(error, 'Unable to retrieve asset plan'));
        });
    }

    function createAssetPlan(req: AssetPlanCreateRequest): Promise<AssetPlanInfoResponse> {
        return services.createAssetPlan(req).then(response => {
            const data = response.data;

            if (!data?.success || !data.result) {
                return Promise.reject({ message: 'Unable to create asset plan' });
            }

            assetPlan.value = data.result;
            return data.result;
        }).catch(error => {
            logger.error('failed to create asset plan', error);
            return Promise.reject(getRequestError(error, 'Unable to create asset plan'));
        });
    }

    function modifyAssetPlan(req: AssetPlanModifyRequest): Promise<AssetPlanInfoResponse> {
        return services.modifyAssetPlan(req).then(response => {
            const data = response.data;

            if (!data?.success || !data.result) {
                return Promise.reject({ message: 'Unable to save asset plan' });
            }

            assetPlan.value = data.result;
            return data.result;
        }).catch(error => {
            logger.error('failed to save asset plan', error);
            return Promise.reject(getRequestError(error, 'Unable to save asset plan'));
        });
    }

    function deleteAssetPlan(): Promise<boolean> {
        return services.deleteAssetPlan().then(response => {
            const data = response.data;

            if (!data?.success || !data.result) {
                return Promise.reject({ message: 'Unable to delete asset plan' });
            }

            assetPlan.value = null;
            return true;
        }).catch(error => {
            logger.error('failed to delete asset plan', error);
            return Promise.reject(getRequestError(error, 'Unable to delete asset plan'));
        });
    }

    function resetAssetPlan(): void {
        assetPlan.value = null;
    }

    return {
        assetPlan,
        loadAssetPlan,
        createAssetPlan,
        modifyAssetPlan,
        deleteAssetPlan,
        resetAssetPlan
    };
});
