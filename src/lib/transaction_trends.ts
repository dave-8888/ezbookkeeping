import { TransactionType } from '@/core/transaction.ts';
import { Transaction, type TransactionListByMaxTimeRequest, type TransactionInfoPageWrapperResponse } from '@/models/transaction.ts';
import type { TransactionListFilter } from '@/stores/transaction.ts';
import { parseDateTimeFromUnixTimeWithTimezoneOffset } from '@/lib/datetime.ts';

export interface DailyTransactionTrend {
    date: string;
    income: number | null;
    expense: number | null;
}

export function buildDailyTransactionTrends(transactions: Transaction[], convert: (transaction: Transaction) => number | null | undefined) {
    const days = new Map<string, DailyTransactionTrend>();
    const seen = new Set<string>();
    let excluded = 0;
    for (const transaction of transactions) {
        if (!transaction.isCashFlow || seen.has(transaction.id)) continue;
        seen.add(transaction.id);
        const date = parseDateTimeFromUnixTimeWithTimezoneOffset(transaction.time, transaction.utcOffset).getGregorianCalendarYearDashMonthDashDay();
        const day = days.get(date) || { date, income: 0, expense: 0 };
        days.set(date, day);
        const field = transaction.type === TransactionType.Income ? 'income' : 'expense';
        const amount = convert(transaction);
        // A gap must not masquerade as a zero or reveal a hidden amount via the line height.
        if (transaction.hideAmount || typeof amount !== 'number' || !Number.isFinite(amount)) {
            day[field] = null;
            if (!transaction.hideAmount) excluded++;
        } else if (day[field] !== null) {
            day[field] += amount;
        }
    }
    const items = [...days.values()].sort((a, b) => a.date.localeCompare(b.date));
    return { items, excluded };
}

export function fillDailyTrendGaps(items: DailyTransactionTrend[], start: string, end: string): DailyTransactionTrend[] {
    if (!items.length) return [];
    const byDate = new Map(items.map(item => [item.date, item]));
    const first = items[0]!.date;
    const last = items[items.length - 1]!.date;
    const cursor = new Date(`${start && start < first ? start : first}T00:00:00Z`);
    const max = end && end > last ? end : last;
    const result: DailyTransactionTrend[] = [];
    while (cursor.toISOString().slice(0, 10) <= max) {
        const date = cursor.toISOString().slice(0, 10);
        result.push(byDate.get(date) || { date, income: 0, expense: 0 });
        cursor.setUTCDate(cursor.getUTCDate() + 1);
    }
    return result;
}

// Keep a snapshot of every filter and avoid mutating the list store while fetching chart pages.
export async function loadTrendTransactions(filter: TransactionListFilter,
    fetchPage: (request: TransactionListByMaxTimeRequest) => Promise<TransactionInfoPageWrapperResponse>,
    isActive: () => boolean): Promise<Transaction[]> {
    const snapshot = { ...filter };
    const items: Transaction[] = [];
    for (let page = 1; isActive(); page++) {
        const result = await fetchPage({ ...snapshot,
            maxTime: snapshot.maxTime > 0 ? snapshot.maxTime * 1000 + 999 : 0,
            minTime: snapshot.minTime * 1000, count: 50, page, withCount: false });
        if (!isActive()) return [];
        items.push(...Transaction.ofMulti(result.items));
        if (result.items.length < 50) break;
    }
    return items;
}
