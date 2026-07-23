import { beforeEach, describe, expect, jest, test } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

import type { AssetPlanInfoResponse } from '@/models/asset_plan.ts';
import services from '@/lib/services.ts';
import { useAssetPlanStore } from '@/stores/assetPlan.ts';

jest.mock('@/lib/services.ts', () => ({
    __esModule: true,
    default: {
        getAssetPlan: jest.fn()
    }
}));

jest.mock('@/lib/logger.ts', () => ({
    __esModule: true,
    default: {}
}));

describe('asset plan store state isolation', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    test('clears user-specific plan data when the root state is reset', () => {
        const store = useAssetPlanStore();
        store.assetPlan = {
            targetAmount: 200,
            targetDate: '2030-01-02',
            startAmount: 100,
            startDate: '2030-01-01',
            currency: 'CNY',
            excludedAccountIds: [],
            currentActualNetWorth: 100,
            todayPlannedNetWorth: 100,
            differenceAmount: 0,
            completionPercentage: '0.00',
            status: 'in_progress',
            comparisonStatus: 'on_track'
        } satisfies AssetPlanInfoResponse;

        store.resetAssetPlan();

        expect(store.assetPlan).toBeNull();
    });

    test('reloads server-derived values whenever the page is entered', async () => {
        const store = useAssetPlanStore();
        const getAssetPlan = jest.mocked(services.getAssetPlan);
        getAssetPlan.mockResolvedValue({
            data: {
                success: true,
                result: null
            }
        } as never);

        await store.loadAssetPlan();
        await store.loadAssetPlan();

        expect(getAssetPlan).toHaveBeenCalledTimes(2);
    });
});
