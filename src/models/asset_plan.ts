export type AssetPlanStatus = 'in_progress' | 'achieved' | 'not_achieved';
export type AssetPlanComparisonStatus = 'ahead' | 'behind' | 'on_track';

export interface AssetPlanCreateRequest {
    readonly targetAmount: number;
    readonly targetDate: string;
    readonly excludedAccountIds: string[];
}

export interface AssetPlanModifyRequest {
    readonly targetAmount: number;
    readonly targetDate: string;
}

export interface AssetPlanInfoResponse {
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
