<template>
    <div class="asset-plan-chart-wrapper">
        <v-chart
            v-if="hasData"
            autoresize
            class="asset-plan-chart"
            :option="chartOptions"
        />
        <div class="asset-plan-chart-empty" v-else>
            <span>{{ tt('No asset trend data') }}</span>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { EChartsOption } from 'echarts';

import { useI18n } from '@/locales/helpers.ts';

import type { AssetPlanInfoResponse } from '@/models/asset_plan.ts';
import {
    type AssetPlanTrendPoint,
    dateOnlyToTimestamp,
    getPlanTrendPoints
} from '@/lib/asset_plan.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';

interface TooltipItem {
    readonly marker: string;
    readonly seriesName: string;
    readonly value: [number, number];
}

const props = defineProps<{
    actualPoints: AssetPlanTrendPoint[];
    currency: string;
    plan?: AssetPlanInfoResponse | null;
    showPlan?: boolean;
}>();

const {
    tt,
    formatDateTimeToLongDate,
    formatAmountToLocalizedNumeralsWithCurrency
} = useI18n();

const planPoints = computed<AssetPlanTrendPoint[]>(() => {
    if (!props.showPlan || !props.plan) {
        return [];
    }

    return getPlanTrendPoints(
        props.plan.startDate,
        props.plan.startAmount,
        props.plan.targetDate,
        props.plan.targetAmount
    );
});

const hasData = computed<boolean>(() => props.actualPoints.length > 0 || planPoints.value.length > 0);

function toChartData(points: AssetPlanTrendPoint[]): [number, number][] {
    return points
        .map(item => [dateOnlyToTimestamp(item.date), item.amount] as [number, number])
        .filter(item => Number.isFinite(item[0]));
}

const chartOptions = computed(() => ({
    animationDuration: 350,
    aria: {
        enabled: true,
        decal: {
            show: true
        }
    },
    color: ['#0071e3', '#ff9f0a'],
    grid: {
        top: 44,
        right: 24,
        bottom: 44,
        left: 24,
        containLabel: true
    },
    legend: {
        top: 4,
        data: props.showPlan && props.plan
            ? [tt('Actual'), tt('Plan')]
            : [tt('Actual')]
    },
    tooltip: {
        trigger: 'axis',
        confine: true,
        formatter: (rawItems: unknown) => {
            const items = (Array.isArray(rawItems) ? rawItems : [rawItems]) as TooltipItem[];
            const first = items[0];

            if (!first?.value) {
                return '';
            }

            const date = formatDateTimeToLongDate(parseDateTimeFromUnixTime(Math.floor(first.value[0] / 1000)));
            const lines = items.map(item => `${item.marker}${item.seriesName}: ${formatAmountToLocalizedNumeralsWithCurrency(item.value[1], props.currency)}`);
            return `${date}<br/>${lines.join('<br/>')}`;
        }
    },
    xAxis: {
        type: 'time',
        axisLabel: {
            hideOverlap: true
        }
    },
    yAxis: {
        type: 'value',
        scale: true,
        axisLabel: {
            formatter: (value: number) => formatAmountToLocalizedNumeralsWithCurrency(Math.trunc(value), props.currency)
        }
    },
    series: [
        {
            id: 'actual',
            name: tt('Actual'),
            type: 'line',
            showSymbol: false,
            smooth: 0.2,
            lineStyle: {
                width: 3
            },
            data: toChartData(props.actualPoints)
        },
        ...(props.showPlan && props.plan ? [{
            id: 'plan',
            name: tt('Plan'),
            type: 'line',
            showSymbol: true,
            symbolSize: 8,
            lineStyle: {
                type: 'dashed' as const,
                width: 3
            },
            data: toChartData(planPoints.value)
        }] : [])
    ]
}) as EChartsOption);
</script>

<style scoped>
.asset-plan-chart-wrapper,
.asset-plan-chart {
    width: 100%;
    min-height: 360px;
}

.asset-plan-chart-empty {
    display: flex;
    min-height: 300px;
    align-items: center;
    justify-content: center;
    color: rgba(var(--v-theme-on-surface), 0.6);
    text-align: center;
}

@media (max-width: 600px) {
    .asset-plan-chart-wrapper,
    .asset-plan-chart {
        min-height: 300px;
    }
}
</style>
