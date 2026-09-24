export type AssetPlanStatus = 'in_progress' | 'achieved' | 'not_achieved';
export type AssetPlanComparisonStatus = 'ahead' | 'behind' | 'on_track';

export interface AssetPlanCreateRequest extends Partial<MonthlyAssetPlanSchedule> {
    readonly targetAmount?: number;
    readonly targetDate?: string;
    readonly excludedAccountIds: string[];
}

export interface AssetPlanModifyRequest extends Partial<MonthlyAssetPlanSchedule> {
    readonly targetAmount?: number;
    readonly targetDate?: string;
}

export interface AssetPlanInfoResponse {
    readonly monthly?: MonthlyAssetPlanResponse;
    readonly targetAmount: number;
    readonly targetDate: string;
    readonly startAmount: number;
    readonly startDate: string;
    readonly currency: string;
    readonly excludedAccountIds: string[];
    readonly currentActualNetWorth: number;
    readonly todayPlannedNetWorth: number;
    readonly differenceAmount: number;
    readonly completionPercentage: string;
    readonly status: AssetPlanStatus;
    readonly comparisonStatus: AssetPlanComparisonStatus;
    readonly targetDateActualNetWorth?: number;
}

export interface AssetPlanGoal { id: string; name: string; amount: number; date: string }
export interface AssetPlanExpense { name: string; amount: number; date: string; goalId?: string }

export interface AssetPlanStage {
    name?: string;
    startDate?: string;
    endDate?: string;
    depositDay?: number;
    goalId?: string;
    amount: number;
    repeat: number;
}
export interface MonthlyAssetPlanSchedule {
    version?: number;
    finalGoal?: AssetPlanGoal;
    goals?: AssetPlanGoal[];
    expenses?: AssetPlanExpense[];
    startMonth: string;
    endMonth: string;
    stages: AssetPlanStage[];
}
export interface AssetPlanMonth {
    plannedToDate?: number;
    income?: number;
    expense?: number;
    month: string;
    planned: number;
    cumulativePlanned: number;
    actual: number | null;
    cumulativeActual: number | null;
    difference: number | null;
    actualThrough?: string;
}
export interface MonthlyAssetPlanResponse extends MonthlyAssetPlanSchedule {
    tracking?: { date: string; amount: number; snapshots: Record<string, number> };
    plannedToDate?: number;
    months: AssetPlanMonth[];
    days?: AssetPlanMonth[];
    projectionBase?: number | null;
    currentNetWorth?: number | null;
    baselineDate: string;
    baselineAmount: number | null;
    asOfDate: string;
    totalPlanned: number;
}
