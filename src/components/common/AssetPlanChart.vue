<template>
    <div class="asset-plan-chart" role="img" :aria-label="tt('Cumulative planned and actual savings')">
        <v-chart v-if="months.length" autoresize class="chart" :option="options" />
        <p v-else>{{ tt('Set your monthly amounts to preview the plan') }}</p>
    </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import type { EChartsOption } from 'echarts';
import type { AssetPlanMonth, AssetPlanGoal, AssetPlanExpense } from '@/models/asset_plan.ts';
import { useI18n } from '@/locales/helpers.ts';
const props = defineProps<{ months: AssetPlanMonth[]; currency: string; dark?: boolean; baseline?: number | null; target?: AssetPlanGoal; goals?: AssetPlanGoal[]; expenses?: AssetPlanExpense[] }>();
const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const absolute = computed(() => props.baseline !== undefined && props.baseline !== null);
const plannedName = computed(() => tt(absolute.value ? 'Expected net worth' : 'Cumulative planned savings'));
const actualName = computed(() => tt(absolute.value ? 'Actual net worth' : 'Cumulative actual savings'));
const goalMarkers = computed(() => (props.goals || []).flatMap(goal => {
    const index = props.months.findIndex(month => goal.date.startsWith(month.month));
    return index < 0 ? [] : [{
        name: goal.name,
        value: [index, props.months[index]!.cumulativePlanned + (props.baseline || 0)],
        goalDate: goal.date,
        goalAmount: goal.amount
    }];
}));
const options = computed<EChartsOption>(() => ({
    animationDuration: 250,
    color: ['#ff9f0a', '#2997ff'],
    grid: { top: 50, right: 20, bottom: 35, left: 15, containLabel: true },
    legend: { top: 0, show: true, data: [plannedName.value, actualName.value, ...(goalMarkers.value.length ? [tt('Short term goals')] : []), ...((props.expenses || []).some(expense => props.months.some(month => expense.date.startsWith(month.month))) ? [tt('Expected expenses')] : []), ...(props.target && absolute.value ? [tt('Final net worth goal')] : [])], textStyle: { color: props.dark ? '#ddd' : '#555' } },
    tooltip: {
        trigger: 'axis', renderMode: 'richText', confine: true,
        formatter: (params: unknown) => {
            const entries = params as { dataIndex: number; seriesName: string; value: unknown; axisValueLabel?: string; marker?: string }[];
            if (!Array.isArray(entries) || !entries.length) return '';
            const row = props.months[entries[0]!.dataIndex];
            if (!row) return '';
            const amount = (value: unknown) => typeof value === 'number' ? formatAmountToLocalizedNumeralsWithCurrency(value, props.currency) : '—';
            return [
                entries[0]!.axisValueLabel || row.month,
                ...entries.map(entry => `${entry.marker || ''}${entry.seriesName}${entry.seriesName === actualName.value && row.actualThrough ? ` (${tt('As of')} ${row.actualThrough})` : ''}  ${amount(entry.value)}`),
                `${tt('Expected net increase')}  ${amount(row.planned)}`
            ].join('\n');
        }
    },
    xAxis: { type: 'category', data: props.months.map(month => month.month), axisLabel: { hideOverlap: true, color: props.dark ? '#aaa' : '#666' } },
    yAxis: { type: 'value', axisLabel: { color: props.dark ? '#aaa' : '#666', formatter: (value: number) => formatAmountToLocalizedNumeralsWithCurrency(value, props.currency) }, splitLine: { lineStyle: { color: props.dark ? '#444' : '#e5e5e5' } } },
    series: [
        { name: plannedName.value, type: 'line', symbolSize: 7, lineStyle: { width: 3, type: 'dashed' }, data: props.months.map(month => month.cumulativePlanned + (props.baseline || 0)) },
        { name: actualName.value, type: 'line', symbolSize: 7, connectNulls: false, lineStyle: { width: 3 }, data: props.months.map(month => month.cumulativeActual === null ? null : month.cumulativeActual + (props.baseline || 0)) },
        ...(props.target && absolute.value ? [{ name: tt('Final net worth goal'), type: 'line' as const, symbol: 'none', itemStyle: { color: '#8e8e93' }, lineStyle: { width: 1, type: 'dotted' as const, color: '#8e8e93' }, data: props.months.map(() => props.target!.amount) }] : []),
        {
            name: tt('Short term goals'), type: 'scatter', symbol: 'diamond', symbolSize: 16, z: 6,
            itemStyle: { color: '#af52de', borderColor: props.dark ? '#202022' : '#fff', borderWidth: 2, opacity: 1 },
            data: goalMarkers.value,
            label: { show: true, position: 'top', distance: 10, color: props.dark ? '#e3b6fa' : '#8137a7', fontSize: 12, width: 150, overflow: 'truncate', formatter: '{b}' },
            labelLayout: { hideOverlap: true },
            emphasis: { scale: 1.3, label: { show: true } },
            tooltip: {
                trigger: 'item', renderMode: 'richText', confine: true,
                formatter: (params: unknown) => {
                    const item = params as { data?: { name: string; goalDate: string; goalAmount: number } };
                    const goal = item.data;
                    return goal ? `${goal.name}\n${goal.goalDate}\n${tt('Target amount')}: ${formatAmountToLocalizedNumeralsWithCurrency(goal.goalAmount, props.currency)}` : '';
                }
            }
        },
        {
            name: tt('Expected expenses'), type: 'scatter', symbol: 'triangle', symbolRotate: 180,
            symbolSize: 14, symbolOffset: [0, 18], z: 6,
            itemStyle: { color: '#ff6961', borderColor: props.dark ? '#202022' : '#fff', borderWidth: 1, opacity: 1 },
            data: (props.expenses || []).flatMap(expense => {
                const index = props.months.findIndex(m => expense.date.startsWith(m.month));
                return index < 0 ? [] : [{
                    value: [index, props.months[index]!.cumulativePlanned + (props.baseline || 0)],
                    name: expense.name, expenseDate: expense.date, expenseAmount: expense.amount
                }];
            }),
            tooltip: {
                trigger: 'item', renderMode: 'richText', confine: true,
                formatter: (params: unknown) => {
                    const item = params as { data?: { name: string; expenseDate: string; expenseAmount: number } };
                    const expense = item.data;
                    return expense ? `${expense.name}\n${expense.expenseDate}\n${tt('Expected expenses')}: ${formatAmountToLocalizedNumeralsWithCurrency(expense.expenseAmount, props.currency)}` : '';
                }
            }
        }
    ]
}));
</script>
<style scoped>
.asset-plan-chart { width: 100%; min-height: 320px; }
.chart { width: 100%; height: 360px; }
.asset-plan-chart > p { padding: 100px 20px; text-align: center; opacity: .65; }
@media (max-width: 600px) { .chart { height: 300px; } }
</style>
