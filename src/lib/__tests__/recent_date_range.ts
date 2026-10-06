import { describe, expect, test } from '@jest/globals';
import moment from 'moment-timezone';
import { getRecentDateRange } from '@/lib/recent_date_range.ts';

const labels = (dates: Date[]) => dates.map(date => moment(date).format('YYYY-MM-DD HH:mm:ss'));

describe('recent custom date ranges', () => {
    test('includes today and the previous six days', () => {
        expect(labels(getRecentDateRange(new Date(2026, 0, 3, 12), 'week'))).toEqual(['2025-12-28 00:00:00', '2026-01-03 23:59:59']);
    });
    test('defaults to one calendar month and clamps month-end on leap years', () => {
        expect(labels(getRecentDateRange(new Date(2024, 2, 31), 'month'))).toEqual(['2024-02-29 00:00:00', '2024-03-31 23:59:59']);
        expect(labels(getRecentDateRange(new Date(2026, 9, 6), 'month', 3))).toEqual(['2026-07-06 00:00:00', '2026-10-06 23:59:59']);
    });
    test('supports multiple years and clamps leap day', () => {
        expect(labels(getRecentDateRange(new Date(2024, 1, 29), 'year'))).toEqual(['2023-02-28 00:00:00', '2024-02-29 23:59:59']);
        expect(labels(getRecentDateRange(new Date(2026, 9, 6), 'year', 2))[0]).toBe('2024-10-06 00:00:00');
    });
    test('rejects invalid counts and dates before the supported epoch', () => {
        for (const count of [0, -1, 1.5, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1]) {
            expect(() => getRecentDateRange(new Date(2026, 9, 6), 'month', count)).toThrow();
        }
        expect(() => getRecentDateRange(new Date(2026, 9, 6), 'year', 100)).toThrow('Date is too early');
    });
});
