<template>
    <section class="monthly-plan" :class="{ dark }" :aria-busy="loading || submitting">
        <header class="plan-header">
            <div><h2>{{ tt('Asset Planning') }}</h2><p>{{ tt('Plan monthly savings and compare with your ledger') }}</p></div>
            <div class="actions">
                <button :disabled="loading || submitting || editing" @click="load">{{ tt('Refresh') }}</button>
                <button v-if="monthly && !editing" class="primary" :disabled="loading" @click="beginEdit">{{ tt('Edit Plan') }}</button>
            </div>
        </header>
        <p v-if="error" role="alert" class="message error">{{ tt(error) }} <button v-if="!editing" @click="load">{{ tt('Retry') }}</button></p>
        <p v-if="notice" role="status" class="message">{{ tt(notice) }}</p>
        <p v-if="loading" role="status" class="panel">{{ tt('Loading...') }}</p>
        <template v-else>
            <section v-if="editing" class="panel editor">
                <h3>{{ tt('Plan your goals') }}</h3>
                <p>{{ tt('Plan savings and expenses alongside your real ledger') }}</p>
                <form @submit.prevent="save">
                    <fieldset :disabled="submitting">
                        <div class="range-fields">
                            <label>{{ tt('Start month') }}<input v-model="startMonth" type="month" min="1970-01" required></label>
                            <label>{{ tt('End month') }}<input v-model="endMonth" type="month" :min="startMonth" required></label>
                        </div>
                        <h3>{{ tt('Final net worth goal') }}</h3>
                        <label class="final-amount">{{ tt('Target net worth including existing assets') }} · {{ currency }}<input v-model="finalAmount" type="text" inputmode="decimal" placeholder="0.00" required></label>
                        <p>{{ tt('The final goal is evaluated at the end of the selected month') }}</p>
                        <h3>{{ tt('Short term goals') }}</h3>
                        <p>{{ tt('Goals describe savings purposes and do not add money to the forecast') }}</p>
                        <div v-for="goal in goals" :key="goal.id" class="goal-row">
                            <label>{{ tt('Goal name') }}<input v-model="goal.name" maxlength="80" required></label>
                            <label>{{ tt('Target amount') }}<input v-model="goal.amount" inputmode="decimal" required></label>
                            <label>{{ tt('Target date') }}<input v-model="goal.date" type="date" :min="startMonth + '-01'" :max="lastMonthDate(endMonth)" required></label>
                            <button type="button" @click="removeGoal(goal.id)">{{ tt('Delete') }}</button>
                        </div>
                        <button type="button" :disabled="goals.length >= 120" @click="addGoal">{{ tt('Add short term goal') }}</button>
                        <h3 class="section-title">{{ tt('Savings schedules') }}</h3>
                        <p>{{ tt('Schedules can run together; missing deposit days use the last day of the month') }}</p>
                        <div v-for="(stage, index) in stages" :key="index" class="savings-card">
                            <div class="goal-row">
                                <label>{{ tt('Savings name') }}<input v-model="stage.name" maxlength="80" required></label>
                                <label>{{ tt('Monthly planned savings') }} · {{ currency }}<input v-model="stage.amount" inputmode="decimal" placeholder="0.00" required></label>
                                <label>{{ tt('Linked goal') }}<select v-model="stage.goalId"><option value="">{{ tt('Long term savings') }}</option><option v-for="g in goals" :key="g.id" :value="g.id">{{ g.name }}</option></select></label>
                                <button type="button" @click="stages.splice(index, 1)">{{ tt('Delete') }}</button>
                            </div>
                            <div class="schedule-fields">
                                <label>{{ tt('Start date') }}<input v-model="stage.startDate" type="date" :min="startMonth + '-01'" :max="lastMonthDate(endMonth)" required></label>
                                <label>{{ tt('Monthly deposit day') }}<input v-model.number="stage.depositDay" type="number" min="1" max="31" required></label>
                                <label>{{ tt('Duration method') }}<select v-model="stage.mode"><option value="repeat">{{ tt('Repeat months') }}</option><option value="date">{{ tt('End date') }}</option></select></label>
                                <label v-if="stage.mode === 'repeat'">{{ tt('Repeat months') }}<input v-model.number="stage.repeat" type="number" min="1" max="120" required></label>
                                <label v-else>{{ tt('End date') }}<input v-model="stage.endDate" type="date" :min="stage.startDate" :max="lastMonthDate(endMonth)" required></label>
                            </div>
                        </div>
                        <button type="button" :disabled="stages.length >= 120" @click="addStage">{{ tt('Add savings schedule') }}</button>
                        <h3 class="section-title">{{ tt('Expected expenses') }}</h3>
                        <p>{{ tt('Expected expenses affect the forecast only; actual assets change when recorded in the ledger') }}</p>
                        <div v-for="(expense, index) in expenses" :key="index" class="savings-card">
                            <div class="goal-row">
                                <label>{{ tt('Expense name') }}<input v-model="expense.name" maxlength="80" required></label>
                                <label>{{ tt('Amount') }} · {{ currency }}<input v-model="expense.amount" inputmode="decimal" required></label>
                                <label>{{ tt('Expected date') }}<input v-model="expense.date" type="date" :min="startMonth + '-01'" :max="lastMonthDate(endMonth)" required></label>
                                <button type="button" @click="expenses.splice(index, 1)">{{ tt('Delete') }}</button>
                            </div>
                            <label>{{ tt('Linked goal') }}<select v-model="expense.goalId"><option value="">{{ tt('No linked goal') }}</option><option v-for="g in goals" :key="g.id" :value="g.id">{{ g.name }}</option></select></label>
                        </div>
                        <button type="button" :disabled="expenses.length >= 120" @click="addExpense">{{ tt('Add expected expense') }}</button>
                        <p v-if="validation" class="validation" role="status">{{ tt(validation) }}</p>
                        <template v-else>
                            <p class="preview-total">{{ tt('Planned net increase') }} <strong>{{ formatAmount(preview.at(-1)?.cumulativePlanned) }}</strong> · {{ preview.length }} {{ tt('Months') }}</p>
                            <asset-plan-chart :months="preview" :currency="currency" :dark="dark" />
                        </template>
                        <div class="actions form-actions">
                            <button type="button" @click="editing = false">{{ tt('Cancel') }}</button>
                            <button type="submit" class="primary" :disabled="!!validation">{{ tt(submitting ? 'Saving...' : 'Save') }}</button>
                        </div>
                    </fieldset>
                </form>
            </section>
            <template v-else-if="monthly">
                <section v-if="monthly.finalGoal" class="panel">
                    <h3>{{ tt('Final net worth goal') }}</h3>
                    <div class="summary-grid">
                        <div><p>{{ monthly.finalGoal.date }} · {{ tt('Target net worth') }}</p><strong>{{ formatAmount(monthly.finalGoal.amount) }}</strong></div>
                        <div><p>{{ tt('Expected net worth') }}</p><strong>{{ formatAmount(expectedNetWorth) }}</strong></div>
                        <div><p>{{ tt('Actual net worth') }}</p><strong>{{ formatAmount(actualNetWorth) }}</strong><small>{{ monthly.asOfDate }}</small></div>
                    </div>
                    <p>{{ tt('Net worth growth is measured from the saved balance baseline. Historical values are shown only for dates with balance snapshots.') }}</p>
                    <p>{{ tt('Forecast gap to goal') }}: {{ formatAmount(expectedNetWorth === null ? null : expectedNetWorth - monthly.finalGoal.amount) }}</p>
                    <p v-if="monthly.baselineAmount === null">{{ tt('Future plan forecast uses current net worth as a provisional starting balance') }}</p>
                </section>
                <section v-if="monthly.goals?.length || monthly.expenses?.length" class="panel">
                    <h3>{{ tt('Goals and expected expenses') }}</h3>
                    <div v-for="goal in monthly.goals" :key="goal.id" class="milestone">
                        <div><strong>{{ goal.name }}</strong><small>{{ goal.date }} · {{ tt('Target amount') }} {{ formatAmount(goal.amount) }}</small></div>
                        <div>{{ tt('Planned funding by target date') }} {{ formatAmount(goalPlanned(goal.id, goal.date)) }}<small>{{ tt('Funding gap') }} {{ formatAmount(Math.max(0, goal.amount - goalPlanned(goal.id, goal.date))) }}</small></div>
                    </div>
                    <div v-for="(expense, index) in monthly.expenses" :key="index" class="milestone">
                        <div><strong>{{ expense.name }}</strong><small>{{ expense.date }} · {{ tt('Expected expense') }}<template v-if="expense.goalId"> · {{ monthly.goals?.find(g => g.id === expense.goalId)?.name }}</template></small></div>
                        <strong class="negative">−{{ formatAmount(expense.amount) }}</strong>
                    </div>
                    <p>{{ tt('Goal funding is planned; actual allocation is not inferred from total assets') }}</p>
                </section>
                <div class="summary-grid">
                    <section class="panel"><p>{{ tt('Planned net increase') }}</p><strong>{{ formatAmount(monthly.totalPlanned) }}</strong><small>{{ monthly.tracking ? monthly.baselineDate : monthly.startMonth }} — {{ monthly.endMonth }}</small></section>
                    <section class="panel"><p>{{ tt('Cumulative net worth growth') }}</p><strong>{{ formatAmount(actualGrowth) }}</strong><small>{{ `${tt('As of')} ${monthly.asOfDate}` }}</small></section>
                    <section class="panel"><p>{{ tt('Cumulative difference') }}</p><strong :class="differenceClass(trackingDifference)">{{ formatAmount(trackingDifference) }}</strong><small>{{ tt('Compared with scheduled amounts since the baseline through today') }}</small></section>
                </div>
                <section class="panel">
                    <h3>{{ tt(monthly.version === 2 ? 'Net worth forecast and actual' : 'Cumulative planned and actual savings') }}</h3>
                    <p>{{ tt('Forecast starts at the balance baseline. Missing historical snapshots are left blank; differences compare only amounts scheduled to date.') }}</p>
                    <label>{{ tt('Chart period') }}<select v-model="detailMonth" :disabled="dailyLoading" @change="loadDays"><option value="">{{ tt('Monthly overview') }}</option><option v-for="m in monthly.months" :key="m.month" :value="m.month">{{ m.month }} · {{ tt('Daily detail') }}</option></select></label>
                    <p v-if="dailyLoading" role="status">{{ tt('Loading...') }}</p>
                    <p v-else-if="dailyError" role="alert">{{ tt(dailyError) }} <button @click="loadDays">{{ tt('Retry') }}</button></p>
                    <asset-plan-chart v-else :months="chartRows" :currency="currency" :dark="dark" :baseline="monthly.version === 2 ? monthly.projectionBase : undefined" :target="monthly.finalGoal" :goals="monthly.goals" :expenses="monthly.expenses" />
                </section>
                <section v-if="current" class="panel current-month">
                    <h3>{{ current.month }} · {{ tt('This month') }}</h3>
                    <div class="summary-grid">
                        <div><p>{{ tt('Planned net increase') }}</p><strong>{{ formatAmount(current.planned) }}</strong></div>
                        <div><p>{{ tt('Net worth growth to date') }}</p><strong>{{ formatAmount(current.actual) }}</strong></div>
                        <div><p>{{ tt('Monthly difference') }}</p><strong :class="differenceClass(current.actual === null ? null : current.actual - (current.plannedToDate ?? current.planned))">{{ formatAmount(current.actual === null ? null : current.actual - (current.plannedToDate ?? current.planned)) }}</strong></div>
                    </div>
                </section>
                <section class="panel">
                    <asset-plan-calendar :plan="monthly" :dark="dark" :format-amount="formatAmount" />
                </section>
                <section class="plan-footer">
                    <p>{{ tt('Net worth growth includes balance adjustments and investment gains or losses since the saved baseline.') }}<br>{{ tt('Baseline') }}: {{ monthly.baselineDate }} · {{ formatAmount(monthly.baselineAmount) }}</p>
                    <button class="danger" :disabled="submitting" @click="deleting = true">{{ tt('Delete Plan') }}</button>
                </section>
            </template>
            <section v-else class="panel empty">
                <h3>{{ tt(plan ? 'Switch to a monthly savings plan' : 'Create your monthly savings plan') }}</h3>
                <p>{{ tt(plan ? 'Your previous goal is preserved until you save a new monthly plan.' : 'Choose your months, set amounts and repeat them to preview your savings.') }}</p>
                <p v-if="plan">{{ tt('Previous target') }}: {{ formatAmount(plan.targetAmount) }} · {{ plan.targetDate }}</p>
                <button class="primary" @click="beginEdit">{{ tt(plan ? 'Set up monthly plan' : 'Create Plan') }}</button>
            </section>
            <section v-if="deleting" class="panel" role="alertdialog" :aria-label="tt('Delete Plan')">
                <p>{{ tt('Are you sure you want to delete this asset plan? This action cannot be undone.') }}</p>
                <div class="actions"><button :disabled="submitting" @click="deleting = false">{{ tt('Cancel') }}</button><button class="danger" :disabled="submitting" @click="deletePlan">{{ tt('Delete') }}</button></div>
            </section>
        </template>
    </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { lastMonthDate } from '@/lib/asset_plan_monthly.ts';
import AssetPlanChart from '@/components/common/AssetPlanChart.vue';
import AssetPlanCalendar from '@/components/common/AssetPlanCalendar.vue';
import { useI18n } from '@/locales/helpers.ts';
import { useAssetPlanPageBase } from '@/views/base/AssetPlanPageBase.ts';
defineProps<{ dark?: boolean }>();
const { tt } = useI18n();
const { finalAmount, goals, expenses, addStage, addGoal, removeGoal, addExpense, goalPlanned, detailMonth, dailyRows, dailyLoading, dailyError, loadDays, loading, submitting, editing, error, notice, startMonth, endMonth, stages, plan, monthly, currency, validation, preview, latest, current, formatAmount, load, beginEdit, save, remove } = useAssetPlanPageBase();
const deleting = ref(false);
const chartRows = computed(() => {
    const data = monthly.value;
    return (detailMonth.value ? dailyRows.value : data?.months ?? []).filter(m => !data?.tracking || m.month >= data.baselineDate.slice(0, m.month.length));
});
const actualGrowth = computed(() => monthly.value?.tracking && monthly.value.currentNetWorth != null ? monthly.value.currentNetWorth - monthly.value.tracking.amount : latest.value?.cumulativeActual);
const trackingDifference = computed(() => monthly.value?.tracking && actualGrowth.value != null ? actualGrowth.value - (monthly.value.plannedToDate ?? 0) : latest.value?.difference);
const expectedNetWorth = computed(() => monthly.value?.projectionBase == null ? null : monthly.value.projectionBase + monthly.value.totalPlanned);
const actualNetWorth = computed(() => monthly.value?.currentNetWorth ?? null);
function differenceClass(value: number | null | undefined): string {
    return value === null || value === undefined || value === 0 ? '' : value > 0 ? 'positive' : 'negative';
}
async function deletePlan(): Promise<void> {
    await remove();
    if (!plan.value) deleting.value = false;
}
onMounted(load);
</script>
<style scoped>
.monthly-plan { --panel: #fff; --border: #dedee3; --muted: #68686e; --soft: #f5f5f7; --text: #1d1d1f; color: var(--text); max-width: 1440px; margin: auto; padding: 24px; font-size: 15px; }
.monthly-plan.dark { --panel: #202022; --border: #38383b; --muted: #aaaab0; --soft: #29292c; --text: #f5f5f7; color-scheme: dark; }
.plan-header, .actions, .plan-footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.plan-header { margin-bottom: 24px; }
h2 { font-size: 28px; margin: 0; font-weight: 600; }
h3 { font-size: 19px; margin: 0 0 12px; font-weight: 600; }
p { color: var(--muted); margin: 8px 0 16px; line-height: 1.6; }
.panel { background: var(--panel); border: 1px solid var(--border); border-radius: 14px; padding: 24px; margin-bottom: 20px; }
.summary-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 20px; }
.summary-grid strong { font-size: 25px; font-weight: 600; overflow-wrap: anywhere; }
.summary-grid small, th small { display: block; color: var(--muted); font-size: 12px; margin-top: 8px; font-weight: 400; }
button { border: 1px solid var(--border); background: var(--panel); border-radius: 9px; padding: 10px 18px; font: inherit; color: var(--text); cursor: pointer; min-height: 44px; }
button.primary { background: #0071e3; border-color: #0071e3; color: white; }
button.danger, .error { color: #d63333; }
button:disabled { opacity: .45; cursor: default; }
button:hover:enabled { filter: brightness(.94); }
button:focus-visible, input:focus-visible, .table-scroll:focus-visible { outline: 3px solid #2997ff; outline-offset: 3px; }
fieldset { border: 0; padding: 0; min-width: 0; }
.range-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin: 24px 0; }
label { display: flex; flex-direction: column; gap: 8px; font-size: 13px; color: var(--muted); }
input, select { display: block; box-sizing: border-box; width: 100%; min-width: 0; min-height: 44px; padding: 10px 12px; border: 1px solid var(--border); border-radius: 8px; color: var(--text); background: var(--panel); font: inherit; font-size: 16px; }
.section-title { margin-top: 32px; }
.final-amount { max-width: 480px; }
.goal-row, .schedule-fields { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)) auto; gap: 16px; align-items: end; margin-bottom: 16px; }
.schedule-fields { grid-template-columns: repeat(4, minmax(0, 1fr)); margin-bottom: 0; }
.savings-card { background: var(--soft); padding: 18px; border-radius: 10px; margin-bottom: 16px; }
.milestone { display: flex; justify-content: space-between; gap: 20px; padding: 16px 0; border-bottom: 1px solid var(--border); }
.milestone small { display: block; color: var(--muted); margin-top: 6px; }
.stage { display: grid; grid-template-columns: 80px minmax(0, 2fr) minmax(0, 1fr) auto; gap: 16px; align-items: end; background: var(--soft); border-radius: 10px; padding: 16px; margin-bottom: 14px; }
.stage > strong { align-self: center; }
.stage button { margin-bottom: 0; }
.validation { color: #a46600; }
.dark .validation, .dark .negative { color: #ffb340; }
.preview-total { margin-top: 24px; }
.form-actions { justify-content: flex-end; margin-top: 24px; }
.message { padding: 14px; background: var(--soft); border-radius: 10px; }
.positive { color: #17804b; }
.dark .positive { color: #55c58b; }
.negative { color: #a46600; }
.table-scroll { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; white-space: nowrap; font-variant-numeric: tabular-nums; }
th, td { padding: 16px 12px; text-align: right; border-bottom: 1px solid var(--border); font-size: 14px; }
th:first-child { text-align: left; }
thead th { color: var(--muted); font-weight: 500; }
.is-current { background: var(--soft); }
.plan-footer { align-items: flex-start; font-size: 13px; }
.plan-footer p { max-width: 850px; }
.empty { text-align: center; padding: 60px 24px; }
@media (max-width: 760px) {
    .monthly-plan { padding: 16px; }
    .summary-grid { grid-template-columns: 1fr; gap: 12px; }
    .summary-grid .panel { margin-bottom: 0; }
    .summary-grid { margin-bottom: 16px; }
    .panel { padding: 18px; }
    .goal-row, .schedule-fields { grid-template-columns: 1fr 1fr; }
    .milestone { flex-direction: column; gap: 8px; }
    .stage { grid-template-columns: 1fr 1fr; }
    .stage > strong { grid-column: 1 / -1; }
    .range-fields { gap: 12px; }
    h2 { font-size: 24px; }
    .plan-header { align-items: flex-start; }
}
</style>

<style scoped>
@media (max-width: 420px) { .goal-row, .schedule-fields, .range-fields { grid-template-columns: 1fr; } }
</style>
