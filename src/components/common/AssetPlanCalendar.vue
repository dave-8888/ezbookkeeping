<template>
    <section ref="calendar" class="asset-calendar" :class="{ dark }" :aria-label="tt('Asset calendar')">
        <header class="calendar-toolbar">
            <div><h3>{{ tt('Asset calendar') }}</h3><p>{{ tt('Explore your plan by year, month and day') }}</p></div>
            <div class="view-switch" role="group" :aria-label="tt('Calendar view')">
                <button :aria-pressed="view === 'year'" @click="view = 'year'">{{ tt('Year view') }}</button>
                <button :aria-pressed="view === 'month'" @click="view = 'month'">{{ tt('Month view') }}</button>
            </div>
        </header>
        <div class="calendar-toolbar period-toolbar">
            <div class="period-controls">
                <button :disabled="!canMove(-1)" :aria-label="tt('Previous period')" @click="move(-1)">‹</button>
                <label><span class="sr-only">{{ tt(view === 'year' ? 'Year view' : 'Month view') }}</span>
                    <select v-if="view === 'year'" v-model="year"><option v-for="y in years" :key="y" :value="y">{{ y }}</option></select>
                    <select v-else v-model="month"><option v-for="m in plan.months" :key="m.month" :value="m.month">{{ m.month }}</option></select>
                </label>
                <button :disabled="!canMove(1)" :aria-label="tt('Next period')" @click="move(1)">›</button>
                <button :disabled="!hasToday" @click="goToday">{{ tt('This month') }}</button>
            </div>
            <div class="legend"><span class="income">{{ tt('Planned savings') }}</span><span class="expense">{{ tt('Expected expenses') }}</span><span class="actual">{{ tt('Net worth growth') }}</span></div>
        </div>
        <div v-if="view === 'year'" class="year-grid">
            <button v-for="cell in yearMonths" :key="cell.key" class="month-card" :disabled="!cell.row" :class="{ today: cell.key === plan.asOfDate.slice(0, 7) }"
                    :title="cell.row ? tt('Double-click to view {year}-{month}', { year, month: cell.index + 1 }) : tt('Outside plan period')"
                    @dblclick="openMonth(cell.key)" @click="onMonthClick($event, cell.key)">
                <span class="month-heading"><strong>{{ monthNames[cell.index] }}</strong><small v-if="cell.key === plan.asOfDate.slice(0, 7)">{{ tt('This month') }}</small></span>
                <template v-if="cell.row">
                    <span class="card-total"><small>{{ tt('Planned net increase') }}</small><strong>{{ formatAmount(cell.row.planned) }}</strong></span>
                    <span v-if="plan.version === 2" class="card-line income"><span>{{ tt('Planned savings') }}</span><b>{{ formatAmount(cell.row.income) }}</b></span>
                    <span v-if="plan.version === 2" class="card-line expense"><span>{{ tt('Expected expenses') }}</span><b>{{ formatAmount(cell.row.expense) }}</b></span>
                    <span class="card-line actual"><span>{{ tt('Net worth growth') }}</span><b>{{ formatAmount(cell.row.actual) }}</b></span>
                    <span class="card-cumulative">
                        <span class="card-line"><span>{{ tt('Cumulative planned net increase') }}</span><b>{{ formatAmount(cell.row.cumulativePlanned) }}</b></span>
                        <span class="card-line"><span>{{ tt('Cumulative net worth growth') }}</span><b>{{ formatAmount(cell.row.cumulativeActual) }}</b></span>
                        <span class="card-line"><span>{{ tt('Cumulative difference') }}</span><b>{{ formatAmount(cell.row.difference) }}</b></span>
                    </span>
                    <small v-if="cell.row.actualThrough" class="as-of">{{ tt('As of') }} {{ cell.row.actualThrough }}</small>
                </template>
                <span v-else class="outside-plan">{{ tt('Outside plan period') }}</span>
            </button>
        </div>
        <template v-else>
            <p class="calendar-note">{{ tt('Amounts follow scheduled dates. Actual growth is shown only where balance data is available.') }}</p>
            <p v-if="plan.version !== 2" class="calendar-note">{{ tt('Legacy monthly plans place savings on the last day of each month.') }}</p>
            <p v-if="loading" role="status">{{ tt('Loading...') }}</p>
            <p v-else-if="error" role="alert">{{ tt(error) }} <button @click="loadDays">{{ tt('Retry') }}</button></p>
            <template v-else>
                <div class="calendar-scroll" tabindex="0" :aria-label="tt('Month view')">
                    <div class="weekday-grid"><span v-for="day in weekdays" :key="day">{{ day }}</span></div>
                    <div class="day-grid">
                        <template v-for="(date, index) in dates" :key="date || index">
                            <button v-if="date" class="day-cell" :class="{ today: date === plan.asOfDate, selected: date === selectedDate }" :aria-pressed="date === selectedDate" :aria-label="date" @click="selectedDate = date">
                                <span class="day-number">{{ Number(date.slice(8)) }}<small v-if="date === plan.asOfDate">{{ tt('Today') }}</small></span>
                                <span v-if="rowsByDate.get(date)?.income" class="day-amount income">{{ tt('Planned savings') }}<b>+{{ formatAmount(rowsByDate.get(date)?.income) }}</b></span>
                                <span v-if="rowsByDate.get(date)?.expense" class="day-amount expense">{{ tt('Expected expenses') }}<b>−{{ formatAmount(rowsByDate.get(date)?.expense) }}</b></span>
                                <span v-if="rowsByDate.get(date)?.actual != null" class="day-amount actual">{{ tt('Net worth growth') }}<b>{{ formatAmount(rowsByDate.get(date)?.actual) }}</b></span>
                                <span v-else-if="rowsByDate.get(date)?.cumulativeActual != null" class="day-amount actual">{{ tt('Balance snapshot') }}</span>
                            </button>
                            <div v-else class="blank-day" aria-hidden="true"></div>
                        </template>
                    </div>
                </div>
                <section class="day-detail" aria-live="polite">
                    <h4>{{ selectedDate }} · {{ tt('Daily detail') }}</h4>
                    <div class="detail-metrics">
                        <div><small>{{ tt('Planned net increase') }}</small><strong>{{ formatAmount(selectedRow?.planned) }}</strong></div>
                        <div class="actual"><small>{{ tt('Net worth growth') }}</small><strong>{{ formatAmount(selectedRow?.actual) }}</strong></div>
                        <div><small>{{ tt('Cumulative difference') }}</small><strong>{{ formatAmount(selectedRow?.difference) }}</strong></div>
                    </div>
                    <ul v-if="selectedEvents.length" class="event-list">
                        <li v-for="(event, index) in selectedEvents" :key="index" :class="event.kind"><span><small>{{ tt(event.kind === 'income' ? 'Planned savings' : 'Expected expenses') }}</small>{{ event.name }}</span><strong>{{ event.kind === 'income' ? '+' : '−' }}{{ formatAmount(event.amount) }}</strong></li>
                    </ul>
                    <p v-else>{{ tt('No scheduled items for this day') }}</p>
                    <p v-if="selectedRow?.actual == null">{{ tt('Daily actual growth is unavailable without consecutive balance snapshots; it is not zero.') }}</p>
                </section>
            </template>
        </template>
    </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import type { AssetPlanMonth, MonthlyAssetPlanResponse } from '@/models/asset_plan.ts';
import { calendarDates, calendarEvents } from '@/lib/asset_plan_calendar.ts';
import services from '@/lib/services.ts';
import { useI18n } from '@/locales/helpers.ts';

const props = defineProps<{ dark?: boolean; plan: MonthlyAssetPlanResponse; formatAmount: (amount: number | null | undefined) => string }>();
const { tt, getAllLongMonthNames, getAllShortWeekdayNames } = useI18n();
const calendar = ref<HTMLElement | null>(null);
const view = ref<'year' | 'month'>('year');
const initialMonth = (): string => props.plan.months.find(m => m.month === props.plan.asOfDate.slice(0, 7))?.month || props.plan.startMonth;
const month = ref(initialMonth());
const year = ref(month.value.slice(0, 4));
const selectedDate = ref('');
const loading = ref(false);
const error = ref('');
const rows = ref<AssetPlanMonth[]>([]);
let requestId = 0;
const years = computed(() => [...new Set(props.plan.months.map(m => m.month.slice(0, 4)))]);
const monthNames = computed(() => getAllLongMonthNames());
const weekdays = computed(() => { const names = getAllShortWeekdayNames(); return [...names.slice(1), names[0]]; });
const yearMonths = computed(() => Array.from({ length: 12 }, (_, index) => {
    const key = `${year.value}-${String(index + 1).padStart(2, '0')}`;
    return { key, index, row: props.plan.months.find(m => m.month === key) };
}));
const dates = computed(() => calendarDates(month.value));
const rowsByDate = computed(() => new Map(rows.value.map(row => [row.month, row])));
const selectedRow = computed(() => rowsByDate.value.get(selectedDate.value));
const events = computed(() => calendarEvents(props.plan));
const selectedEvents = computed(() => events.value.filter(event => event.date === selectedDate.value));
const hasToday = computed(() => props.plan.months.some(m => m.month === props.plan.asOfDate.slice(0, 7)));
function canMove(offset: number): boolean {
    const periods = view.value === 'year' ? years.value : props.plan.months.map(m => m.month);
    const index = periods.indexOf(view.value === 'year' ? year.value : month.value) + offset;
    return index >= 0 && index < periods.length;
}
function move(offset: number): void {
    if (!canMove(offset)) return;
    if (view.value === 'year') year.value = years.value[years.value.indexOf(year.value) + offset]!;
    else month.value = props.plan.months[props.plan.months.findIndex(m => m.month === month.value) + offset]!.month;
}
function onKeydown(event: KeyboardEvent): void {
    if (event.defaultPrevented || event.isComposing || event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return;
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    if (!calendar.value?.getClientRects().length) return;

    const target = event.target;
    if (target instanceof HTMLElement && (target.isContentEditable || target.closest(
        'input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="textbox"], [role="combobox"], [role="slider"], [role="spinbutton"], [role="tablist"], [role="menu"], [role="listbox"]'
    ))) return;

    // Let dialogs and popovers keep their own keyboard navigation.
    const overlays = document.querySelectorAll('.v-overlay--active, .popup.modal-in, .popover.modal-in, [role="dialog"], [role="alertdialog"], dialog[open]');
    if (Array.from(overlays).some(overlay => overlay.getClientRects().length > 0)) return;

    const offset = event.key === 'ArrowLeft' ? -1 : 1;
    if (!canMove(offset)) return;
    event.preventDefault();
    move(offset);
}
function openMonth(value: string): void { month.value = value; view.value = 'month'; }
function onMonthClick(event: MouseEvent, value: string): void {
    // Keyboard and assistive technology clicks have no pointer click count.
    if (event.detail === 0) openMonth(value);
}
function goToday(): void { month.value = initialMonth(); year.value = month.value.slice(0, 4); }
async function loadDays(): Promise<void> {
    const id = ++requestId;
    rows.value = []; error.value = ''; loading.value = true;
    try {
        const response = await services.getAssetPlan(month.value);
        if (!response.data?.success || !response.data.result?.monthly?.days) throw new Error('Unable to retrieve asset plan');
        if (id === requestId) rows.value = response.data.result.monthly.days;
    } catch {
        if (id === requestId) error.value = 'Unable to retrieve asset plan';
    } finally { if (id === requestId) loading.value = false; }
}
watch(year, value => {
    if (month.value.slice(0, 4) !== value) month.value = props.plan.months.find(m => m.month.startsWith(value))!.month;
});
watch([month, view, () => props.plan], () => {
    if (!props.plan.months.some(m => m.month === month.value)) month.value = initialMonth();
    year.value = month.value.slice(0, 4);
    selectedDate.value = month.value === props.plan.asOfDate.slice(0, 7) ? props.plan.asOfDate : `${month.value}-01`;
    if (view.value === 'month') void loadDays();
    else { requestId++; loading.value = false; }
}, { immediate: true });
onMounted(() => window.addEventListener('keydown', onKeydown));
onBeforeUnmount(() => {
    requestId++;
    window.removeEventListener('keydown', onKeydown);
});
</script>

<style scoped>
.asset-calendar { --income: #087648; --income-bg: #eaf7ef; --expense: #a64c12; --expense-bg: #fff2e5; --actual: #0065c7; --actual-bg: #eaf3ff; }
.asset-calendar.dark { --income: #71dfaa; --income-bg: #17392c; --expense: #ffbc7d; --expense-bg: #422e20; --actual: #85bcff; --actual-bg: #20354b; }
h3 { margin: 0; font-size: 19px; font-weight: 600; }
h4 { margin: 0 0 20px; font-size: 16px; }
p { color: var(--muted); font-size: 13px; line-height: 1.6; margin: 8px 0 0; }
button, select { font: inherit; color: var(--text); background: var(--panel); border: 1px solid var(--border); border-radius: 9px; min-height: 44px; padding: 8px 14px; }
button { cursor: pointer; }
button:disabled { cursor: default; opacity: .45; }
button:focus-visible, select:focus-visible, .calendar-scroll:focus-visible { outline: 3px solid #2997ff; outline-offset: 3px; }
.calendar-toolbar, .period-controls, .legend { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.view-switch { display: flex; border-radius: 11px; padding: 3px; background: var(--soft); }
.view-switch button { background: transparent; border-color: transparent; min-width: 88px; }
.view-switch button[aria-pressed="true"] { background: var(--panel); box-shadow: 0 1px 5px #00000015; color: var(--actual); }
.period-toolbar { margin: 24px 0 20px; }
.period-controls { justify-content: flex-start; gap: 8px; }
.legend { gap: 8px; font-size: 12px; }
.legend span { border-radius: 6px; padding: 6px 9px; }
.income { color: var(--income); }
.expense { color: var(--expense); }
.actual { color: var(--actual); }
.legend .income, .day-amount.income { background: var(--income-bg); }
.legend .expense, .day-amount.expense { background: var(--expense-bg); }
.legend .actual, .day-amount.actual { background: var(--actual-bg); }
.year-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
.month-card { display: flex; flex-direction: column; text-align: start; padding: 20px; min-width: 0; border-radius: 14px; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; transition: border-color .15s, box-shadow .15s; }
.month-card:hover:enabled { border-color: var(--actual); box-shadow: 0 3px 14px #0000000a; }
.month-heading { display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; }
.month-heading > strong { font-size: 18px; }
.month-heading small { color: var(--actual); font-size: 11px; }
.card-total { display: flex; flex-direction: column; gap: 6px; margin: 20px 0 16px; }
.card-total small, .card-cumulative, .as-of { color: var(--muted); }
.card-total strong { font-size: 23px; font-weight: 600; }
.card-line { display: flex; justify-content: space-between; gap: 12px; width: 100%; font-size: 12px; padding: 5px 0; }
.card-line b { font-weight: 500; text-align: end; }
.card-cumulative { border-top: 1px solid var(--border); padding-top: 10px; margin-top: 12px; width: 100%; }
.as-of { font-size: 11px; margin-top: 10px; }
.outside-plan { margin: auto 0; padding: 40px 0; font-size: 13px; }
.today { border-color: var(--actual); }
.calendar-note { margin: 0 0 16px; }
.calendar-scroll { overflow-x: auto; padding: 3px; }
.weekday-grid, .day-grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); min-width: 630px; gap: 6px; }
.weekday-grid { text-align: center; font-size: 12px; color: var(--muted); margin-bottom: 12px; }
.day-cell { min-height: 138px; padding: 9px; text-align: start; display: flex; flex-direction: column; gap: 5px; border-radius: 10px; font-variant-numeric: tabular-nums; }
.day-cell:hover { background: var(--soft); }
.day-cell.selected { outline: 2px solid var(--actual); outline-offset: -2px; background: var(--actual-bg); }
.day-number { display: flex; gap: 4px; align-items: center; margin-bottom: 5px; }
.day-number small { font-size: 10px; color: var(--actual); }
.day-amount { display: flex; flex-direction: column; padding: 5px; font-size: 10px; border-radius: 5px; overflow-wrap: anywhere; }
.day-amount b { font-size: 12px; font-weight: 500; }
.blank-day { background: var(--soft); border-radius: 10px; opacity: .5; }
.day-detail { background: var(--soft); margin-top: 20px; padding: 20px; border-radius: 12px; }
.detail-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
.detail-metrics small, .detail-metrics strong { display: block; overflow-wrap: anywhere; }
.detail-metrics small { font-size: 12px; margin-bottom: 6px; }
.detail-metrics strong { font-size: 18px; font-weight: 600; }
.event-list { list-style: none; padding: 0; margin: 16px 0 0; }
.event-list li { display: flex; justify-content: space-between; gap: 16px; padding: 12px 0; border-top: 1px solid var(--border); overflow-wrap: anywhere; }
.event-list small { display: block; margin-bottom: 4px; font-size: 11px; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
@media (min-width: 1400px) { .year-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
@media (max-width: 1000px) { .year-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 520px) {
    .year-grid, .detail-metrics { grid-template-columns: 1fr; }
    .calendar-toolbar { align-items: flex-start; }
    .month-card { padding: 18px; }
    .weekday-grid, .day-grid { min-width: 0; gap: 4px; }
    .day-cell { min-height: 78px; padding: 6px; }
    .day-number { font-size: 12px; }
    .day-number small { display: none; }
    .day-amount { font-size: 0; height: 6px; padding: 0; border-radius: 3px; }
    .day-amount b { font-size: 0; }
    .day-amount.income { background: var(--income); }
    .day-amount.expense { background: var(--expense); }
    .day-amount.actual { background: var(--actual); }
}
</style>
