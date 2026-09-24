import { computed, ref } from 'vue';
import { useI18n } from '@/locales/helpers.ts';
import { useAssetPlanStore } from '@/stores/assetPlan.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { lastMonthDate, planEvents, monthAt, monthIndex, parseMonthlyAmount, previewMonthlyPlan, validateMonthlyPlan } from '@/lib/asset_plan_monthly.ts';

export function useAssetPlanPageBase() {
    const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
    const store = useAssetPlanStore();
    const settings = useSettingsStore();
    const user = useUserStore();
    const loading = ref(false);
    const submitting = ref(false);
    const editing = ref(false);
    const error = ref('');
    const notice = ref('');
    const startMonth = ref('');
    const endMonth = ref('');
    const stages = ref<{ name: string; amount: string; repeat: number; startDate: string; endDate: string; depositDay: number; goalId: string; mode: string }[]>([]);
    const finalAmount = ref('');
    const goals = ref<{ id: string; name: string; amount: string; date: string }[]>([]);
    const expenses = ref<{ name: string; amount: string; date: string; goalId: string }[]>([]);
    const detailMonth = ref('');
    const dailyLoading = ref(false);
    const dailyError = ref('');
    const dailyRows = ref<import('@/models/asset_plan.ts').AssetPlanMonth[]>([]);
    const plan = computed(() => store.assetPlan);
    const monthly = computed(() => plan.value?.monthly);
    const currency = computed(() => plan.value?.currency || user.currentUserDefaultCurrency);
    const schedule = computed(() => ({ version: 2, startMonth: startMonth.value, endMonth: endMonth.value,
        finalGoal: { id: 'final', name: 'Final goal', amount: parseMonthlyAmount(finalAmount.value), date: lastMonthDate(endMonth.value) },
        goals: goals.value.map(goal => ({ ...goal, amount: parseMonthlyAmount(goal.amount) })),
        expenses: expenses.value.map(expense => ({ ...expense, amount: parseMonthlyAmount(expense.amount) })),
        stages: stages.value.map(stage => ({ name: stage.name, amount: parseMonthlyAmount(stage.amount), repeat: stage.mode === 'repeat' ? Number(stage.repeat) : 0, startDate: stage.startDate, endDate: stage.mode === 'date' ? stage.endDate : '', depositDay: Number(stage.depositDay), goalId: stage.goalId })) }));
    const validation = computed(() => validateMonthlyPlan(schedule.value));
    const preview = computed(() => previewMonthlyPlan(schedule.value));
    const latest = computed(() => monthly.value?.months.filter(month => month.cumulativeActual !== null).at(-1));
    const current = computed(() => monthly.value?.months.find(month => month.month === monthly.value?.asOfDate.slice(0, 7)));

    function formatAmount(amount: number | null | undefined): string {
        return amount === null || amount === undefined ? '—' : formatAmountToLocalizedNumeralsWithCurrency(amount, currency.value);
    }
    function setError(value: unknown): void {
        const err = value as { message?: string };
        error.value = err?.message || 'An error occurred';
    }
    async function load(): Promise<void> {
        loading.value = true;
        error.value = '';
        try { await store.loadAssetPlan(); detailMonth.value = ''; dailyRows.value = []; } catch (err) { setError(err); } finally { loading.value = false; }
    }
    function beginEdit(): void {
        const now = new Date();
        startMonth.value = monthly.value?.startMonth || `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
        endMonth.value = monthly.value?.endMonth || monthAt(monthIndex(startMonth.value) + 11);
        let offset = 0;
        stages.value = monthly.value?.stages.map((stage, index) => {
            const startDate = stage.startDate || `${monthAt(monthIndex(startMonth.value) + offset)}-01`;
            offset += stage.repeat;
            return { name: stage.name || `${tt('Savings')} ${index + 1}`, amount: (stage.amount / 100).toFixed(2), repeat: stage.repeat || 1, startDate, endDate: stage.endDate || lastMonthDate(endMonth.value), depositDay: stage.depositDay || 31, goalId: stage.goalId || '', mode: stage.endDate ? 'date' : 'repeat' };
        }) || [{ name: tt('Long term savings'), amount: '', repeat: 12, startDate: `${startMonth.value}-01`, endDate: lastMonthDate(endMonth.value), depositDay: 1, goalId: '', mode: 'repeat' }];
        finalAmount.value = monthly.value?.finalGoal ? (monthly.value.finalGoal.amount / 100).toFixed(2) : (!monthly.value && plan.value ? (plan.value.targetAmount / 100).toFixed(2) : '');
        goals.value = (monthly.value?.goals || []).map(g => ({ ...g, amount: (g.amount / 100).toFixed(2) }));
        expenses.value = (monthly.value?.expenses || []).map(e => ({ ...e, goalId: e.goalId || '', amount: (e.amount / 100).toFixed(2) }));
        error.value = '';
        notice.value = '';
        editing.value = true;
    }
    async function save(): Promise<void> {
        if (validation.value || submitting.value) return;
        submitting.value = true;
        error.value = '';
        try {
            if (plan.value) { await store.modifyAssetPlan(schedule.value); } else {
                await store.createAssetPlan({ ...schedule.value, excludedAccountIds: Object.keys(settings.appSettings.statistics.defaultAccountFilter || {}).filter(id => settings.appSettings.statistics.defaultAccountFilter[id]) });
            }
            editing.value = false;
            detailMonth.value = ''; dailyRows.value = [];
            notice.value = 'Asset plan has been saved';
        } catch (err) { setError(err); } finally { submitting.value = false; }
    }
    async function remove(): Promise<void> {
        if (submitting.value) return;
        submitting.value = true;
        error.value = '';
        try { await store.deleteAssetPlan(); editing.value = false; notice.value = 'Asset plan has been deleted'; } catch (err) { setError(err); } finally { submitting.value = false; }
    }
    function addStage(): void { stages.value.push({ name: '', amount: '', repeat: 1, startDate: `${startMonth.value}-01`, endDate: lastMonthDate(endMonth.value), depositDay: 1, goalId: '', mode: 'repeat' }); }
    function addGoal(): void { goals.value.push({ id: crypto.randomUUID(), name: '', amount: '', date: lastMonthDate(endMonth.value) }); }
    function removeGoal(id: string): void {
        goals.value = goals.value.filter(g => g.id !== id);
        for (const stage of stages.value) if (stage.goalId === id) stage.goalId = '';
        for (const expense of expenses.value) if (expense.goalId === id) expense.goalId = '';
    }
    function addExpense(): void { expenses.value.push({ name: '', amount: '', date: lastMonthDate(endMonth.value), goalId: '' }); }
    function goalPlanned(id: string, date: string): number {
        return monthly.value ? planEvents(monthly.value).filter(e => e.goalId === id && e.amount > 0 && e.date <= date).reduce((total, e) => total + e.amount, 0) : 0;
    }
    async function loadDays(): Promise<void> {
        dailyError.value = ''; dailyRows.value = [];
        if (!detailMonth.value) return;
        dailyLoading.value = true;
        try { const result = await store.loadAssetPlan(detailMonth.value); dailyRows.value = result?.monthly?.days || []; }
        catch (err) { dailyError.value = (err as { message?: string })?.message || 'An error occurred'; }
        finally { dailyLoading.value = false; }
    }
    return { finalAmount, goals, expenses, addStage, addGoal, removeGoal, addExpense, goalPlanned, detailMonth, dailyRows, dailyLoading, dailyError, loadDays, loading, submitting, editing, error, notice, startMonth, endMonth, stages, plan, monthly, currency, validation, preview, latest, current, formatAmount, load, beginEdit, save, remove };
}
