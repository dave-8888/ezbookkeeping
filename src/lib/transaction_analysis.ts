import { TransactionType } from '@/core/transaction.ts';
import type { Transaction } from '@/models/transaction.ts';

export interface ExpenseRankingItem {
    key: string;
    amount: number;
    transaction: Transaction;
}

export function buildExpenseRankings(transactions: Transaction[], convert: (transaction: Transaction) => number | null | undefined) {
    const days = new Map<string, ExpenseRankingItem>();
    const categories = new Map<string, ExpenseRankingItem>();
    const singles: ExpenseRankingItem[] = [];
    let excluded = 0;
    const seen = new Set<string>();
    for (const transaction of transactions) {
        if (transaction.type !== TransactionType.Expense || seen.has(transaction.id)) continue;
        seen.add(transaction.id);
        const amount = convert(transaction);
        if (typeof amount !== 'number' || !Number.isFinite(amount)) {
            excluded++;
            continue;
        }
        singles.push({ key: transaction.id, amount, transaction });
        for (const [map, key] of [
            [days, transaction.gregorianCalendarYearDashMonthDashDay || ''],
            [categories, transaction.categoryId || '0']
        ] as const) {
            const existing = map.get(key);
            if (existing) existing.amount += amount;
            else map.set(key, { key, amount, transaction });
        }
    }
    const sort = (items: ExpenseRankingItem[]) => items.sort((a, b) => b.amount - a.amount || a.key.localeCompare(b.key));
    return { days: sort([...days.values()]), categories: sort([...categories.values()]), singles: sort(singles), excluded };
}
