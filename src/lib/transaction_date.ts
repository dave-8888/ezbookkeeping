import moment from 'moment-timezone';

// Keep the transaction's wall-clock time and fixed UTC offset when changing its day.
export function moveTransactionToDate(time: number, utcOffset: number, date: string): number {
    const target = moment.utc(date, 'YYYY-MM-DD', true);
    if (!target.isValid()) {
        throw new Error('Invalid transaction date');
    }

    const original = moment.unix(time).utcOffset(utcOffset);
    return target.utcOffset(utcOffset, true).set({
        hour: original.hour(), minute: original.minute(), second: original.second()
    }).unix();
}
