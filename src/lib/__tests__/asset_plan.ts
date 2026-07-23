import { describe, expect, test } from '@jest/globals';

import type { TransactionAssetTrendsAnalysisData } from '@/models/transaction.ts';
import {
    dateOnlyToTimestamp,
    getActualTrendPoints,
    getPlanTrendPoints,
    validateAssetPlanInput
} from '@/lib/asset_plan.ts';

describe('asset plan input validation', () => {
    test('requires a target above the frozen starting amount', () => {
        expect(validateAssetPlanInput(10000, '2030-01-02', 10000, '2030-01-01'))
            .toBe('Target amount must be greater than the plan starting amount');
        expect(validateAssetPlanInput(10001, '2030-01-02', 10000, '2030-01-01'))
            .toBeNull();
    });

    test('requires a valid future natural date', () => {
        expect(validateAssetPlanInput(20000, '2030-01-01', 10000, '2030-01-01'))
            .toBe('Target date must be later than today');
        expect(validateAssetPlanInput(20000, '2030-02-30', 10000, '2030-01-01'))
            .toBe('Target date must be later than today');
        expect(Number.isNaN(dateOnlyToTimestamp('2030-02-30'))).toBe(true);
    });
});

describe('asset plan chart data', () => {
    test('keeps the frozen start and target as the plan line endpoints', () => {
        expect(getPlanTrendPoints('2030-01-01', 10000, '2031-01-01', 20000)).toStrictEqual([
            { date: '2030-01-01', amount: 10000 },
            { date: '2031-01-01', amount: 20000 }
        ]);
    });

    test('combines account series into one actual trend without mixing dates', () => {
        const data = {
            items: [
                {
                    items: [
                        { year: 2030, month: 1, day: 2, totalAmount: 100 },
                        { year: 2030, month: 1, day: 1, totalAmount: 80 }
                    ]
                },
                {
                    items: [
                        { year: 2030, month: 1, day: 1, totalAmount: 20 },
                        { year: 2030, month: 1, day: 2, totalAmount: 30 }
                    ]
                }
            ]
        } as TransactionAssetTrendsAnalysisData;

        expect(getActualTrendPoints(data)).toStrictEqual([
            { date: '2030-01-01', amount: 100 },
            { date: '2030-01-02', amount: 130 }
        ]);
    });
});
