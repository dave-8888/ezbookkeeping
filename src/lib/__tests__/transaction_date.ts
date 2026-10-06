import { describe, expect, test } from '@jest/globals';
import { moveTransactionToDate } from '@/lib/transaction_date.ts';

describe('move transaction to another day', () => {
    test.each([
        ['2026-10-06T14:20:37+08:00', 480, '2026-09-30', '2026-09-30T14:20:37+08:00'],
        ['2026-01-01T00:15:03+14:00', 840, '2025-12-31', '2025-12-31T00:15:03+14:00'],
        ['2026-10-06T23:59:59-07:00', -420, '2024-02-29', '2024-02-29T23:59:59-07:00'],
        ['2026-10-06T10:30:00+05:45', 345, '2026-10-06', '2026-10-06T10:30:00+05:45']
    ])('preserves time and offset: %s', (original, offset, date, expected) => {
        expect(moveTransactionToDate(Date.parse(original) / 1000, offset, date)).toBe(Date.parse(expected) / 1000);
    });

    test.each(['2026-02-29', '2026-13-01', 'invalid'])('rejects invalid date %s', date => {
        expect(() => moveTransactionToDate(0, 480, date)).toThrow('Invalid transaction date');
    });
});
