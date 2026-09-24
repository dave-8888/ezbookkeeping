import { describe, expect, test } from '@jest/globals';
import { parseMonthlyAmount, previewMonthlyPlan, validateMonthlyPlan } from '@/lib/asset_plan_monthly.ts';

describe('monthly savings rehearsal', () => {
    test('expands stages across the year without including existing assets', () => {
        const months = previewMonthlyPlan({ startMonth: '2026-11', endMonth: '2027-02', stages: [{ amount: 500000, repeat: 2 }, { amount: 800000, repeat: 2 }] });
        expect(months.map(month => month.month)).toEqual(['2026-11', '2026-12', '2027-01', '2027-02']);
        expect(months.map(month => month.cumulativePlanned)).toEqual([500000, 1000000, 1800000, 2600000]);
        expect(months.every(month => month.actual === null)).toBe(true);
    });
    test('rejects incomplete schedules and unsafe totals', () => {
        expect(validateMonthlyPlan({ startMonth: '2026-01', endMonth: '2026-02', stages: [{ amount: 100, repeat: 1 }] })).not.toBeNull();
        expect(validateMonthlyPlan({ startMonth: '2026-01', endMonth: '2026-02', stages: [{ amount: Number.MAX_SAFE_INTEGER, repeat: 2 }] })).not.toBeNull();
        expect(validateMonthlyPlan({ startMonth: '2026-01', endMonth: '2026-02', stages: [{ amount: 0, repeat: 1 }, { amount: 100, repeat: 1 }] })).toBeNull();
    });
    test('parses currency without floating-point cent truncation', () => {
        expect(parseMonthlyAmount('32.82')).toBe(3282);
        expect(parseMonthlyAmount('0.29')).toBe(29);
        expect(parseMonthlyAmount('1.001')).toBeNaN();
        expect(parseMonthlyAmount('')).toBeNaN();
        expect(parseMonthlyAmount('-1')).toBeNaN();
    });
});

describe('goals and expected expenses', () => {
    const schedule = {
        version: 2, startMonth: '2026-10', endMonth: '2026-12',
        finalGoal: { id: 'final', name: 'Final', amount: 3000000, date: '2026-12-31' },
        goals: [{ id: 'phone', name: 'Phone', amount: 1200000, date: '2026-12-20' }],
        stages: [
            { name: 'Long term', amount: 1000000, repeat: 3, startDate: '2026-10-01', depositDay: 1 },
            { name: 'Phone', amount: 400000, repeat: 3, startDate: '2026-10-01', depositDay: 1, goalId: 'phone' }
        ],
        expenses: [{ name: 'Phone', amount: 1200000, date: '2026-12-20', goalId: 'phone' }]
    };
    test('counts parallel savings once and deducts consumption', () => {
        expect(validateMonthlyPlan(schedule)).toBeNull();
        expect(previewMonthlyPlan(schedule).map(m => m.cumulativePlanned)).toEqual([1400000, 2800000, 3000000]);
        expect(previewMonthlyPlan({ ...schedule, goals: [{ ...schedule.goals[0]!, amount: 9000000 }] }).at(-1)?.cumulativePlanned).toBe(3000000);
    });
    test('rejects dangling goal links and schedules beyond the range', () => {
        expect(validateMonthlyPlan({ ...schedule, goals: [] })).not.toBeNull();
        expect(validateMonthlyPlan({ ...schedule, stages: [{ ...schedule.stages[0]!, repeat: 4 }] })).not.toBeNull();
        expect(validateMonthlyPlan({ ...schedule, expenses: [{ ...schedule.expenses[0]!, date: '2026-12-32' }] })).not.toBeNull();
    });
    test('clamps deposit dates in leap February and respects an inclusive end date', () => {
        const calendar = { ...schedule, startMonth: '2024-01', endMonth: '2024-03', finalGoal: { ...schedule.finalGoal, date: '2024-03-31' }, goals: [], expenses: [], stages: [{ name: 'Savings', amount: 100, repeat: 0, startDate: '2024-01-31', endDate: '2024-03-30', depositDay: 31 }] };
        expect(validateMonthlyPlan(calendar)).toBeNull();
        expect(previewMonthlyPlan(calendar).map(m => m.planned)).toEqual([100, 100, 0]);
    });
});
