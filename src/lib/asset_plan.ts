import type {
    TransactionAssetTrendsAnalysisData,
    TransactionAssetTrendsAnalysisDataItem
} from '@/models/transaction.ts';

export type AssetPlanMetric = 'net_worth' | 'total_assets' | 'total_liabilities';

export interface AssetPlanTrendPoint {
    readonly date: string;
    readonly amount: number;
}

function padDatePart(value: number): string {
    return value.toString().padStart(2, '0');
}

export function formatLocalDate(value: Date): string {
    return `${value.getFullYear()}-${padDatePart(value.getMonth() + 1)}-${padDatePart(value.getDate())}`;
}

export function getTodayLocalDate(): string {
    return formatLocalDate(new Date());
}

export function dateOnlyToTimestamp(value: string): number {
    const parts = value.split('-').map(part => Number.parseInt(part, 10));

    if (parts.length !== 3 || parts.some(part => !Number.isInteger(part))) {
        return Number.NaN;
    }

    const date = new Date(parts[0] as number, (parts[1] as number) - 1, parts[2] as number);

    if (formatLocalDate(date) !== value) {
        return Number.NaN;
    }

    return date.getTime();
}

export function validateAssetPlanInput(targetAmount: number, targetDate: string, startAmount: number, today: string = getTodayLocalDate()): string | null {
    if (!Number.isSafeInteger(targetAmount) || targetAmount <= startAmount) {
        return 'Target amount must be greater than the plan starting amount';
    }

    const targetTimestamp = dateOnlyToTimestamp(targetDate);
    const todayTimestamp = dateOnlyToTimestamp(today);

    if (!Number.isFinite(targetTimestamp) || !Number.isFinite(todayTimestamp) || targetTimestamp <= todayTimestamp) {
        return 'Target date must be later than today';
    }

    return null;
}

export function getPlanTrendPoints(startDate: string, startAmount: number, targetDate: string, targetAmount: number): AssetPlanTrendPoint[] {
    if (!Number.isFinite(dateOnlyToTimestamp(startDate)) || !Number.isFinite(dateOnlyToTimestamp(targetDate))) {
        return [];
    }

    return [
        { date: startDate, amount: startAmount },
        { date: targetDate, amount: targetAmount }
    ];
}

export function getActualTrendPoints(data: TransactionAssetTrendsAnalysisData | null): AssetPlanTrendPoint[] {
    if (!data?.items?.length) {
        return [];
    }

    const values: Record<string, number> = {};

    for (const series of data.items as TransactionAssetTrendsAnalysisDataItem[]) {
        for (const item of series.items) {
            const date = `${item.year}-${padDatePart(item.month)}-${padDatePart(item.day)}`;
            values[date] = (values[date] ?? 0) + item.totalAmount;
        }
    }

    return Object.entries(values)
        .map(([date, amount]) => ({ date, amount }))
        .sort((first, second) => first.date.localeCompare(second.date));
}
