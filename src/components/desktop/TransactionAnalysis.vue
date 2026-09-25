<template>
    <div class="pa-6" :aria-busy="loading">
        <v-skeleton-loader v-if="loading" type="article, article, article" />
        <template v-else>
            <v-alert v-if="rankings.excluded" type="warning" variant="tonal" class="mb-4">
                {{ tt('Expenses without exchange rates are excluded from rankings') }} ({{ rankings.excluded }})
            </v-alert>
            <v-row>
                <v-col v-for="section in sections" :key="section.title" cols="12" lg="4">
                    <v-card variant="outlined" class="ranking-card">
                        <v-card-title class="text-subtitle-1 px-4 pt-4 pb-3">{{ tt(section.title) }}</v-card-title>
                        <v-card-text v-if="!section.items.length">{{ tt('No transaction data') }}</v-card-text>
                        <ol v-else class="ranking-list px-4 pb-4">
                            <li v-for="(item, index) in section.items" :key="item.key" class="mb-4">
                                <component :is="section.single ? 'button' : 'div'" class="ranking-item"
                                           :type="section.single ? 'button' : undefined"
                                           @click="section.single && emit('select', item.transaction)">
                                    <div class="d-flex align-start ga-2">
                                        <span class="text-medium-emphasis">{{ index + 1 }}.</span>
                                        <span class="ranking-label">{{ section.label(item) }}</span>
                                        <span class="text-expense text-no-wrap">{{ visible(item, section.single) ? formatAmountToLocalizedNumeralsWithCurrency(Math.trunc(item.amount), defaultCurrency) : '***' }}</span>
                                    </div>
                                    <div v-if="section.single" class="text-caption text-medium-emphasis mt-1">{{ dateLabel(item) }}</div>
                                    <v-progress-linear v-if="visible(item, section.single)" class="mt-2" rounded height="10" color="primary"
                                                       :model-value="barWidth(item, section.items)" :aria-label="section.label(item)" />
                                </component>
                            </li>
                        </ol>
                    </v-card>
                </v-col>
            </v-row>
        </template>
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from '@/locales/helpers.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import type { Transaction } from '@/models/transaction.ts';
import { buildExpenseRankings, type ExpenseRankingItem } from '@/lib/transaction_analysis.ts';

const props = defineProps<{ transactions: Transaction[]; loading: boolean; defaultCurrency: string; showAmounts: boolean }>();
const emit = defineEmits<{ select: [transaction: Transaction] }>();
const { tt, getWeekdayLongName, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const exchangeRatesStore = useExchangeRatesStore();
const rankings = computed(() => buildExpenseRankings(props.transactions, transaction => {
    const currency = transaction.currency || props.defaultCurrency;
    return currency === props.defaultCurrency ? transaction.sourceAmount :
        exchangeRatesStore.getExchangedAmount(transaction.sourceAmount, currency, props.defaultCurrency);
}));
function dateLabel(item: ExpenseRankingItem): string {
    const transaction = item.transaction;
    return `${transaction.gregorianCalendarYearDashMonthDashDay || ''} ${transaction.displayDayOfWeek !== undefined ? getWeekdayLongName(transaction.displayDayOfWeek) : ''}`;
}
const sections = computed(() => [
    { title: 'Daily Expense Ranking', items: rankings.value.days, single: false, label: dateLabel },
    { title: 'Secondary Category Expense Ranking', items: rankings.value.categories, single: false,
        label: (item: ExpenseRankingItem) => item.transaction.category?.name || tt('Uncategorized') },
    { title: 'Individual Expense Ranking', items: rankings.value.singles, single: true,
        label: (item: ExpenseRankingItem) => item.transaction.comment || item.transaction.category?.name || tt('Expense') }
]);
function visible(item: ExpenseRankingItem, single: boolean): boolean {
    return props.showAmounts && (!single || !item.transaction.hideAmount);
}
function barWidth(item: ExpenseRankingItem, items: ExpenseRankingItem[]): number {
    const max = items[0]?.amount || 0;
    return max > 0 ? Math.max(0, item.amount / max * 100) : 0;
}
</script>

<style scoped>
.ranking-card { height: 100%; }
.ranking-list { list-style: none; max-height: 640px; overflow-y: auto; }
.ranking-item { display: block; width: 100%; text-align: start; color: inherit; }
button.ranking-item { cursor: pointer; border-radius: 4px; }
button.ranking-item:hover { background: rgba(var(--v-theme-on-surface), 0.04); }
.ranking-label { flex: 1; min-width: 0; overflow-wrap: anywhere; }
</style>
