<template>
    <section class="px-6 pb-4" :aria-label="tt(periodTitle)" :aria-busy="loading">
        <div class="d-flex align-center justify-space-between flex-wrap gap-2 mb-2">
            <h2 class="text-subtitle-1">{{ tt(periodTitle) }}</h2>
            <v-btn-toggle v-model="period" mandatory density="compact" variant="outlined" divided
                          :aria-label="tt('Trend interval')">
                <v-btn v-for="option in periods" :key="option.value" :value="option.value" size="small">{{ tt(option.label) }}</v-btn>
            </v-btn-toggle>
            <span class="text-caption text-medium-emphasis">{{ defaultCurrency }}</span>
        </div>
        <div v-if="!showAmounts" class="py-6 text-center text-medium-emphasis">{{ tt('Amounts are hidden') }}</div>
        <v-skeleton-loader v-else-if="loading" type="image" height="260" />
        <v-alert v-else-if="failed" type="error" variant="tonal">
            {{ tt('Unable to retrieve transaction list') }}
            <v-btn variant="text" size="small" @click="retry++">{{ tt('Retry') }}</v-btn>
        </v-alert>
        <template v-else>
            <v-alert v-if="trends.excluded" type="warning" variant="tonal" class="mb-2">
                {{ tt('Trend gaps indicate hidden amounts or missing exchange rates') }}
            </v-alert>
            <div v-if="!points.length" class="py-6 text-center text-medium-emphasis">{{ tt('No transaction data') }}</div>
            <template v-else>
                <v-chart autoresize class="transaction-trends-chart" :option="options" :update-options="{ notMerge: true }" />
                <p class="text-caption text-medium-emphasis mt-1 mb-0">{{ tt('Trend gaps indicate hidden amounts or missing exchange rates') }}</p>
            </template>
        </template>
    </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useTheme } from 'vuetify';
import { use } from 'echarts/core';
import { DataZoomComponent } from 'echarts/components';
import { useI18n } from '@/locales/helpers.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import { useUserStore } from '@/stores/user.ts';
import type { TransactionListFilter } from '@/stores/transaction.ts';
import type { Transaction } from '@/models/transaction.ts';
import { ThemeType } from '@/core/theme.ts';
import { type TransactionTrendPeriod, aggregateTransactionTrends, buildDailyTransactionTrends, fillDailyTrendGaps, loadTrendTransactions } from '@/lib/transaction_trends.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';
import { getExpenseAndIncomeAmountColor } from '@/lib/ui/common.ts';
import services from '@/lib/services.ts';

use([DataZoomComponent]);
const props = defineProps<{ filter: TransactionListFilter; refreshKey: number; defaultCurrency: string; showAmounts: boolean }>();
const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const theme = useTheme();
const userStore = useUserStore();
const exchangeRatesStore = useExchangeRatesStore();
const transactions = ref<Transaction[]>([]);
const loading = ref(true);
const failed = ref(false);
const retry = ref(0);
const period = ref<TransactionTrendPeriod>('day');
const periods = [
    { value: 'day', label: 'Daily', title: 'Daily Income and Expense Trends' },
    { value: 'week', label: 'Weekly', title: 'Weekly Income and Expense Trends' },
    { value: 'month', label: 'Monthly', title: 'Monthly Income and Expense Trends' },
    { value: 'year', label: 'Yearly', title: 'Yearly Income and Expense Trends' }
];
const periodTitle = computed(() => periods.find(option => option.value === period.value)!.title);

watch(() => [JSON.stringify(props.filter), props.refreshKey, props.showAmounts, retry.value], async (_value, _oldValue, onCleanup) => {
    let active = true;
    onCleanup(() => { active = false; });
    transactions.value = [];
    failed.value = false;
    loading.value = props.showAmounts;
    if (!props.showAmounts) return;
    try {
        const items = await loadTrendTransactions(props.filter, async request => {
            const response = await services.getTransactions(request);
            if (!response.data?.success || !response.data.result) throw new Error('Unable to retrieve transaction list');
            return response.data.result;
        }, () => active);
        if (active) transactions.value = items;
    } catch {
        if (active) failed.value = true;
    } finally {
        if (active) loading.value = false;
    }
}, { immediate: true });

const trends = computed(() => buildDailyTransactionTrends(transactions.value, transaction => {
    const currency = transaction.currency || props.defaultCurrency;
    return currency === props.defaultCurrency ? transaction.sourceAmount :
        exchangeRatesStore.getExchangedAmount(transaction.sourceAmount, currency, props.defaultCurrency);
}));
const dailyPoints = computed(() => fillDailyTrendGaps(trends.value.items,
    props.filter.minTime ? parseDateTimeFromUnixTime(props.filter.minTime).getGregorianCalendarYearDashMonthDashDay() : '',
    props.filter.maxTime ? parseDateTimeFromUnixTime(props.filter.maxTime).getGregorianCalendarYearDashMonthDashDay() : ''));
const points = computed(() => aggregateTransactionTrends(dailyPoints.value, period.value, userStore.currentUserFirstDayOfWeek));
const options = computed(() => {
    const colors = getExpenseAndIncomeAmountColor(userStore.currentUserExpenseAmountColor, userStore.currentUserIncomeAmountColor, theme.global.name.value === ThemeType.Dark);
    const textColor = theme.current.value.colors['on-surface'];
    const amount = (value: number | null) => value === null ? null : Math.trunc(value) / 100;
    return {
        animation: false,
        aria: { enabled: true },
        color: [colors.incomeAmountColor, colors.expenseAmountColor],
        textStyle: { color: textColor },
        legend: { data: [tt('Income'), tt('Expense')], textStyle: { color: textColor } },
        grid: { top: 40, left: 12, right: 18, bottom: points.value.length > 60 ? 64 : 16, containLabel: true },
        tooltip: { trigger: 'axis', renderMode: 'richText', valueFormatter: (value: number | null) => value === null ? '—' : formatAmountToLocalizedNumeralsWithCurrency(Math.round(value * 100), props.defaultCurrency) },
        xAxis: { type: 'category', data: points.value.map(day => day.date), boundaryGap: false, axisLabel: { color: textColor } },
        yAxis: { type: 'value', axisLabel: { color: textColor } },
        dataZoom: points.value.length > 60 ? [{ type: 'slider', start: 0, end: 100, height: 20, bottom: 4 }] : [],
        series: [
            { name: tt('Income'), type: 'line', showSymbol: points.value.length <= 60, symbolSize: 6, data: points.value.map(day => amount(day.income)) },
            { name: tt('Expense'), type: 'line', showSymbol: points.value.length <= 60, symbolSize: 6, lineStyle: { type: 'solid' }, data: points.value.map(day => amount(day.expense)) }
        ]
    };
});
</script>

<style scoped>
.transaction-trends-chart { height: 280px; width: 100%; }
</style>
