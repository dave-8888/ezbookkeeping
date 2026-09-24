import { describe, expect, test } from '@jest/globals';
import { WeekDay } from '@/core/datetime.ts';
import { Transaction } from '@/models/transaction.ts';
import { TransactionType } from '@/core/transaction.ts';
import { buildExpenseRankings } from '@/lib/transaction_analysis.ts';

function expense(id: string, amount: number, category: string, date: string): Transaction {
    const transaction = Transaction.of({ id, timeSequenceId: id, type: TransactionType.Expense,
        categoryId: category, time: 0, utcOffset: 0, sourceAccountId: '1', destinationAccountId: '0',
        sourceAmount: amount, destinationAmount: 0, hideAmount: false, tagIds: [], comment: '', editable: true });
    transaction.setDisplayDate(date as Parameters<Transaction['setDisplayDate']>[0], 1, WeekDay.Thursday);
    return transaction;
}

describe('expense rankings', () => {
    test('ranks complete daily, secondary category and individual expenses separately', () => {
        const a = expense('1', 300, 'child-a', '2026-09-24');
        const b = expense('2', 400, 'child-b', '2026-09-24');
        const c = expense('3', 600, 'child-a', '2026-09-23');
        const income = expense('4', 9999, 'income', '2026-09-23');
        income.type = TransactionType.Income;
        const result = buildExpenseRankings([a, b, c, income, a], t => t.sourceAmount);
        expect(result.days.map(i => [i.key, i.amount])).toEqual([['2026-09-24', 700], ['2026-09-23', 600]]);
        expect(result.categories.map(i => [i.key, i.amount])).toEqual([['child-a', 900], ['child-b', 400]]);
        expect(result.singles.map(i => i.key)).toEqual(['3', '2', '1']);
    });
    test('uses converted amounts and reports missing rates without fabricating totals', () => {
        const result = buildExpenseRankings([
            expense('1', 100, 'a', '2026-09-24'), expense('2', 200, 'b', '2026-09-24'),
            expense('3', 300, 'c', '2026-09-24')
        ], t => t.id === '1' ? 700 : t.id === '2' ? 200 : undefined);
        expect(result.singles.map(i => i.amount)).toEqual([700, 200]);
        expect(result.days[0]?.amount).toBe(900);
        expect(result.excluded).toBe(1);
        expect(buildExpenseRankings([], t => t.sourceAmount).singles).toEqual([]);
    });
});
