import moment from 'moment-timezone';

export type RecentDateRangeUnit = 'week' | 'month' | 'year';

// Dates here represent the wall-clock values displayed by the date picker.
// The dialog converts them back into the user's configured timezone on confirmation.
export function getRecentDateRange(today: Date, unit: RecentDateRangeUnit, count = 1): Date[] {
    if (!Number.isSafeInteger(count) || count < 1) throw new Error('Please enter a positive integer');
    const end = moment(today).endOf('day');
    const start = moment(today).startOf('day');
    if (unit === 'week') start.subtract(6, 'days');
    else start.subtract(count, unit);
    if (!start.isValid() || start.year() < 1970) throw new Error('Date is too early');
    return [start.toDate(), end.toDate()];
}
