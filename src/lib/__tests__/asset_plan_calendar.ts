import { describe, expect, test } from '@jest/globals';
import { calendarDates, calendarEvents } from '@/lib/asset_plan_calendar.ts';
import type { MonthlyAssetPlanResponse } from '@/models/asset_plan.ts';

describe('asset plan calendar dates', () => {
    test('aligns Monday first, pads complete weeks and preserves leap day', () => {
        const dates = calendarDates('2024-02');
        expect(dates.slice(0, 4)).toEqual([null, null, null, '2024-02-01']);
        expect(dates.filter(Boolean)).toHaveLength(29);
        expect(dates).toContain('2024-02-29');
        expect(dates.length % 7).toBe(0);
        expect(calendarDates('2026-03')).toHaveLength(42);
        expect(calendarDates('2027-02')).toHaveLength(28);
    });
    test('retains exact dates, parallel deposits and month-end fallback without spreading money', () => {
        const plan: MonthlyAssetPlanResponse = {
            version: 2, startMonth: '2026-01', endMonth: '2026-03',
            stages: [
                { name: 'Savings', amount: 10000, startDate: '2026-01-01', depositDay: 31, repeat: 3 },
                { name: 'Trip', amount: 2000, startDate: '2026-02-01', depositDay: 28, repeat: 1 }
            ],
            expenses: [{ name: 'Ticket', amount: 5000, date: '2026-02-14' }],
            baselineDate: '2026-01-31', baselineAmount: 0, asOfDate: '2026-02-01', totalPlanned: 17000,
            tracking: { date: '2026-01-31', amount: 0, snapshots: {} }, months: []
        };
        const events = calendarEvents(plan);
        expect(events.map(e => e.date)).toEqual(['2026-02-28', '2026-03-31', '2026-02-28', '2026-02-14']);
        expect(events.find(e => e.kind === 'expense')).toMatchObject({ name: 'Ticket', amount: 5000 });
        expect(events.filter(e => e.date === '2026-02-28').reduce((sum, e) => sum + e.amount, 0)).toBe(12000);
        expect(calendarEvents({ ...plan, version: 1 })).toEqual([]);
    });
});
