import { describe, expect, test } from '@jest/globals';
import { Transaction, type TransactionInfoResponse, type TransactionListByMaxTimeRequest } from '@/models/transaction.ts';
import { TransactionType } from '@/core/transaction.ts';
import type { TransactionListFilter } from '@/stores/transaction.ts';
import { aggregateTransactionTrends, buildDailyTransactionTrends, fillDailyTrendGaps, loadTrendTransactions } from '@/lib/transaction_trends.ts';

function transaction(id: string, type = TransactionType.Expense, utcOffset = 0): Transaction {
    return Transaction.of({ id, timeSequenceId: id, type, categoryId: '1',
        time: Date.parse('2026-09-24T23:30:00Z') / 1000, utcOffset, sourceAccountId: '0', destinationAccountId: '0',
        sourceAmount: 100, destinationAmount: 0, hideAmount: false, tagIds: [], comment: '', editable: true });
}
const filter: TransactionListFilter = { dateType: 0, minTime: 10, maxTime: 20, type: 0, categoryIds: '2,3', accountIds: '4', tagFilter: '1:5', amountFilter: 'gt:100', keyword: 'lunch' };

describe('daily transaction trends', () => {
    test('groups using transaction timezone, separates income and expense, deduplicates and excludes transfers', () => {
        const expense = transaction('1');
        const result = buildDailyTransactionTrends([expense, expense, transaction('2', TransactionType.Income, 480), transaction('3', TransactionType.Transfer)], t => t.sourceAmount);
        expect(result.items).toEqual([
            { date: '2026-09-24', income: 0, expense: 100 },
            { date: '2026-09-25', income: 100, expense: 0 }
        ]);
    });
    test('converts currencies and leaves gaps for hidden or unavailable amounts, regardless of order', () => {
        const hidden = transaction('1'); hidden.hideAmount = true;
        const result = buildDailyTransactionTrends([transaction('2'), hidden, transaction('3'), transaction('4', TransactionType.Income)], t => t.id === '4' ? undefined : t.sourceAmount * 7);
        expect(result.items).toEqual([{ date: '2026-09-24', income: null, expense: null }]);
        expect(result.excluded).toBe(1);
        expect(buildDailyTransactionTrends([transaction('2')], t => t.sourceAmount * 7).items[0]?.expense).toBe(700);
    });
    test('fills missing days and range edges, including leap day, without inventing data for an empty result', () => {
        const items = [{ date: '2024-02-28', income: 100, expense: 0 }, { date: '2024-03-01', income: 0, expense: 50 }];
        const result = fillDailyTrendGaps(items, '2024-02-27', '2024-03-02');
        expect(result.map(day => day.date)).toEqual(['2024-02-27', '2024-02-28', '2024-02-29', '2024-03-01', '2024-03-02']);
        expect(result[2]?.income).toBe(0);
        expect(fillDailyTrendGaps([], '2024-02-27', '2024-03-02')).toEqual([]);
    });
    test('loads every page with a stable copy of all filters and exact time boundaries', async () => {
        const requests: TransactionListByMaxTimeRequest[] = [];
        const currentFilter = { ...filter };
        const items = await loadTrendTransactions(currentFilter, async request => {
            requests.push(request);
            currentFilter.keyword = 'changed';
            return { items: Array.from({ length: request.page < 3 ? 50 : 1 }, (_, i) => transaction(`${request.page}-${i}`) as TransactionInfoResponse) };
        }, () => true);
        expect(items).toHaveLength(101);
        expect(requests.map(request => request.page)).toEqual([1, 2, 3]);
        expect(requests[2]).toMatchObject({ ...filter, minTime: 10000, maxTime: 20999, count: 50, keyword: 'lunch' });
    });
    test('discards stale results and stops pagination after a filter change', async () => {
        let active = true;
        let calls = 0;
        const items = await loadTrendTransactions(filter, async () => {
            calls++;
            active = false;
            return { items: Array.from({ length: 50 }, (_, i) => transaction(`${i}`) as TransactionInfoResponse) };
        }, () => active);
        expect(items).toEqual([]);
        expect(calls).toBe(1);
    });
    test('propagates request errors instead of publishing a partial trend', async () => {
        await expect(loadTrendTransactions(filter, async request => {
            if (request.page === 2) throw new Error('offline');
            return { items: Array.from({ length: 50 }, (_, i) => transaction(`${i}`) as TransactionInfoResponse) };
        }, () => true)).rejects.toThrow('offline');
    });
});


describe('transaction trend intervals', () => {
    const items = [
        { date: '2023-12-31', income: 100, expense: 20 },
        { date: '2024-01-01', income: 200, expense: 30 },
        { date: '2024-01-07', income: 300, expense: null },
        { date: '2024-02-01', income: 400, expense: 50 }
    ];
    test('keeps daily results and respects the configured week boundary across years', () => {
        expect(aggregateTransactionTrends(items, 'day', 1)).toEqual(items);
        expect(aggregateTransactionTrends(items, 'week', 1)).toEqual([
            { date: '2023-12-25', income: 100, expense: 20 },
            { date: '2024-01-01', income: 500, expense: null },
            { date: '2024-01-29', income: 400, expense: 50 }
        ]);
        expect(aggregateTransactionTrends(items, 'week', 0)[0]).toEqual({ date: '2023-12-31', income: 300, expense: 50 });
    });
    test('groups calendar months and years while preserving unknown amounts', () => {
        expect(aggregateTransactionTrends(items, 'month', 1)).toEqual([
            { date: '2023-12', income: 100, expense: 20 },
            { date: '2024-01', income: 500, expense: null },
            { date: '2024-02', income: 400, expense: 50 }
        ]);
        expect(aggregateTransactionTrends(items, 'year', 1)).toEqual([
            { date: '2023', income: 100, expense: 20 },
            { date: '2024', income: 900, expense: null }
        ]);
        expect(aggregateTransactionTrends([], 'year', 1)).toEqual([]);
    });
    test('fills empty periods and only sums supplied transactions at partial period edges', () => {
        const days = fillDailyTrendGaps([{ date: '2024-02-29', income: 100, expense: 50 }], '2024-01-20', '2024-03-04');
        expect(aggregateTransactionTrends(days, 'month', 1)).toEqual([
            { date: '2024-01', income: 0, expense: 0 },
            { date: '2024-02', income: 100, expense: 50 },
            { date: '2024-03', income: 0, expense: 0 }
        ]);
    });
});
