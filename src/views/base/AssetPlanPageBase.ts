import { computed, ref } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useAccountsStore } from '@/stores/account.ts';
import { useAssetPlanStore } from '@/stores/assetPlan.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useStatisticsStore } from '@/stores/statistics.ts';
import { useUserStore } from '@/stores/user.ts';

import { DateRange } from '@/core/datetime.ts';
import { ChartDataType, StatisticsAnalysisType } from '@/core/statistics.ts';
import type { AssetPlanInfoResponse } from '@/models/asset_plan.ts';
import {
    type AssetPlanMetric,
    type AssetPlanTrendPoint,
    dateOnlyToTimestamp,
    formatLocalDate,
    getActualTrendPoints,
    getTodayLocalDate,
    validateAssetPlanInput
} from '@/lib/asset_plan.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';

const METRIC_CHART_DATA_TYPES: Record<AssetPlanMetric, number> = {
    net_worth: ChartDataType.NetWorth.type,
    total_assets: ChartDataType.AccountTotalAssets.type,
    total_liabilities: ChartDataType.AccountTotalLiabilities.type
};

function errorMessage(error: unknown): string {
    if (error && typeof error === 'object') {
        const value = error as { message?: string, error?: { errorMessage?: string } };
        return value.message || value.error?.errorMessage || 'An error occurred';
    }

    return typeof error === 'string' ? error : 'An error occurred';
}

export function useAssetPlanPageBase() {
    const {
        tt,
        formatDateTimeToLongDate,
        formatAmountToLocalizedNumeralsWithCurrency,
        formatPercentToLocalizedNumerals
    } = useI18n();

    const accountsStore = useAccountsStore();
    const assetPlanStore = useAssetPlanStore();
    const settingsStore = useSettingsStore();
    const statisticsStore = useStatisticsStore();
    const userStore = useUserStore();

    const loading = ref<boolean>(true);
    const reloading = ref<boolean>(false);
    const submitting = ref<boolean>(false);
    const deleting = ref<boolean>(false);
    const editing = ref<boolean>(false);
    const loadError = ref<string>('');
    const selectedMetric = ref<AssetPlanMetric>('net_worth');
    const targetAmount = ref<number>(0);
    const targetDate = ref<string>('');

    const plan = computed<AssetPlanInfoResponse | null>(() => assetPlanStore.assetPlan);
    const currency = computed<string>(() => plan.value?.currency || userStore.currentUserDefaultCurrency);
    const showPlanDetails = computed<boolean>(() => selectedMetric.value === 'net_worth' && !!plan.value);
    const actualTrendPoints = computed<AssetPlanTrendPoint[]>(() => getActualTrendPoints(statisticsStore.assetTrendsData));
    const currentMetricAmount = computed<number>(() => actualTrendPoints.value.at(-1)?.amount ?? 0);
    const metricName = computed<string>(() => {
        if (selectedMetric.value === 'total_assets') {
            return tt('Total Assets');
        } else if (selectedMetric.value === 'total_liabilities') {
            return tt('Total Liabilities');
        }

        return tt('Net Worth');
    });
    const minimumTargetAmount = computed<number>(() => plan.value?.startAmount ?? currentMetricAmount.value);
    const validationMessage = computed<string | null>(() => {
        if (!editing.value) {
            return null;
        }

        return validateAssetPlanInput(targetAmount.value, targetDate.value, minimumTargetAmount.value);
    });
    const planStatusName = computed<string>(() => {
        if (plan.value?.status === 'achieved') {
            return tt('Goal achieved');
        } else if (plan.value?.status === 'not_achieved') {
            return tt('Goal not achieved');
        }

        return tt('Plan in progress');
    });
    const comparisonStatusName = computed<string>(() => {
        if (plan.value?.comparisonStatus === 'ahead') {
            return tt('Ahead of plan');
        } else if (plan.value?.comparisonStatus === 'behind') {
            return tt('Behind plan');
        }

        return tt('On track');
    });
    const completionDisplay = computed<string>(() => {
        const percentage = Number.parseFloat(plan.value?.completionPercentage || '0');
        return formatPercentToLocalizedNumerals(percentage, 2, '&lt;0.01');
    });

    function formatDisplayAmount(amount: number): string {
        return formatAmountToLocalizedNumeralsWithCurrency(amount, currency.value);
    }

    function formatDisplayDate(value: string): string {
        const timestamp = dateOnlyToTimestamp(value);

        if (!Number.isFinite(timestamp)) {
            return value;
        }

        return formatDateTimeToLongDate(parseDateTimeFromUnixTime(Math.floor(timestamp / 1000)));
    }

    function configureTrends(): void {
        const today = new Date();
        const oneYearAgo = new Date(today.getFullYear() - 1, today.getMonth(), today.getDate());
        const planStartTimestamp = plan.value ? dateOnlyToTimestamp(plan.value.startDate) : Number.NaN;
        const startTimestamp = Number.isFinite(planStartTimestamp)
            ? Math.min(planStartTimestamp, oneYearAgo.getTime())
            : oneYearAgo.getTime();
        const endOfToday = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59);
        const excludedAccountIds = plan.value?.excludedAccountIds
            ?? Object.keys(settingsStore.appSettings.statistics.defaultAccountFilter || {})
                .filter(id => settingsStore.appSettings.statistics.defaultAccountFilter[id]);
        const filterAccountIds: Record<string, boolean> = {};

        for (const id of excludedAccountIds) {
            filterAccountIds[id] = true;
        }

        statisticsStore.initTransactionStatisticsFilter(StatisticsAnalysisType.AssetTrends, {
            chartDataType: METRIC_CHART_DATA_TYPES[selectedMetric.value],
            assetTrendsChartDateType: DateRange.Custom.type,
            assetTrendsChartStartTime: Math.floor(startTimestamp / 1000),
            assetTrendsChartEndTime: Math.floor(endOfToday.getTime() / 1000),
            filterAccountIds
        });
    }

    function loadTrends(force: boolean): Promise<unknown> {
        return statisticsStore.loadAssetTrends({ force }).catch(error => {
            if (error?.isUpToDate) {
                return statisticsStore.assetTrendsData;
            }

            return Promise.reject(error);
        });
    }

    function load(): Promise<void> {
        loading.value = true;
        loadError.value = '';

        return Promise.all([
            accountsStore.loadAllAccounts({ force: false }),
            assetPlanStore.loadAssetPlan({ force: false })
        ]).then(() => {
            configureTrends();
            return loadTrends(false);
        }).then(() => {
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            loadError.value = errorMessage(error);
            return Promise.reject(error);
        });
    }

    function reload(): Promise<void> {
        reloading.value = true;
        loadError.value = '';

        return Promise.all([
            accountsStore.loadAllAccounts({ force: true }),
            assetPlanStore.loadAssetPlan({ force: true })
        ]).then(() => {
            configureTrends();
            return loadTrends(true);
        }).then(() => {
            reloading.value = false;
        }).catch(error => {
            reloading.value = false;
            loadError.value = errorMessage(error);
            return Promise.reject(error);
        });
    }

    function selectMetric(metric: AssetPlanMetric): void {
        selectedMetric.value = metric;
        statisticsStore.updateTransactionStatisticsFilter({
            chartDataType: METRIC_CHART_DATA_TYPES[metric]
        });
    }

    function beginCreate(): void {
        targetAmount.value = Math.max(0, currentMetricAmount.value);
        targetDate.value = formatLocalDate(new Date(new Date().getFullYear() + 1, new Date().getMonth(), new Date().getDate()));
        editing.value = true;
    }

    function beginEdit(): void {
        if (!plan.value) {
            return;
        }

        targetAmount.value = plan.value.targetAmount;
        targetDate.value = plan.value.targetDate;
        editing.value = true;
    }

    function cancelEdit(): void {
        editing.value = false;
    }

    function save(): Promise<AssetPlanInfoResponse> {
        const validationError = validateAssetPlanInput(targetAmount.value, targetDate.value, minimumTargetAmount.value);

        if (validationError) {
            return Promise.reject({ message: validationError });
        }

        submitting.value = true;
        const request = plan.value
            ? assetPlanStore.modifyAssetPlan({
                targetAmount: targetAmount.value,
                targetDate: targetDate.value
            })
            : assetPlanStore.createAssetPlan({
                targetAmount: targetAmount.value,
                targetDate: targetDate.value,
                excludedAccountIds: Object.keys(settingsStore.appSettings.statistics.defaultAccountFilter || {})
                    .filter(id => settingsStore.appSettings.statistics.defaultAccountFilter[id])
            });

        return request.then(result => {
            editing.value = false;
            configureTrends();
            return loadTrends(true).then(() => result);
        }).finally(() => {
            submitting.value = false;
        });
    }

    function remove(): Promise<boolean> {
        deleting.value = true;

        return assetPlanStore.deleteAssetPlan().then(result => {
            editing.value = false;
            configureTrends();
            return loadTrends(true).then(() => result);
        }).finally(() => {
            deleting.value = false;
        });
    }

    return {
        loading,
        reloading,
        submitting,
        deleting,
        editing,
        loadError,
        selectedMetric,
        targetAmount,
        targetDate,
        plan,
        currency,
        showPlanDetails,
        actualTrendPoints,
        currentMetricAmount,
        metricName,
        minimumTargetAmount,
        validationMessage,
        planStatusName,
        comparisonStatusName,
        completionDisplay,
        formatDisplayAmount,
        formatDisplayDate,
        load,
        reload,
        selectMetric,
        beginCreate,
        beginEdit,
        cancelEdit,
        save,
        remove,
        getTodayLocalDate
    };
}
