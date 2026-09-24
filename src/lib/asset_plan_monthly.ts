import type { AssetPlanMonth, MonthlyAssetPlanSchedule } from '@/models/asset_plan.ts';

export function monthIndex(month: string): number {
    if (!/^\d{4}-(0[1-9]|1[0-2])$/.test(month) || Number(month.slice(0, 4)) < 1970) return Number.NaN;
    return Number(month.slice(0, 4)) * 12 + Number(month.slice(5)) - 1;
}

export function monthAt(index: number): string {
    return `${Math.floor(index / 12)}-${String(index % 12 + 1).padStart(2, '0')}`;
}

export function parseMonthlyAmount(value: string): number {
    if (!/^\d+(\.\d{0,2})?$/.test(value)) return Number.NaN;
    const [whole, fraction = ''] = value.split('.');
    const result = Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
    return Number.isSafeInteger(result) ? result : Number.NaN;
}

export function validateMonthlyPlan(schedule: MonthlyAssetPlanSchedule): string | null {
    if (schedule.version === 2) return validateGoalPlan(schedule);
    const start = monthIndex(schedule.startMonth);
    const count = monthIndex(schedule.endMonth) - start + 1;
    if (!Number.isFinite(start) || !Number.isFinite(count) || count < 1 || count > 120) return 'Choose a range of 1 to 120 months';
    let repeats = 0;
    let total = 0;
    for (const stage of schedule.stages) {
        if (!Number.isSafeInteger(stage.amount) || stage.amount < 0 || !Number.isInteger(stage.repeat) || stage.repeat < 1 || stage.repeat > 120) return 'Enter a non-negative amount and a positive repeat count';
        repeats += stage.repeat;
        total += stage.amount * stage.repeat;
    }
    if (repeats !== count) return 'Stage repeats must exactly cover the selected months';
    if (!Number.isSafeInteger(total) || total <= 0) return 'The total planned savings must be positive and within the amount limit';
    return null;
}

export function previewMonthlyPlan(schedule: MonthlyAssetPlanSchedule): AssetPlanMonth[] {
    if (validateMonthlyPlan(schedule)) return [];
    if (schedule.version === 2) return previewGoalPlan(schedule);
    const months: AssetPlanMonth[] = [];
    let total = 0;
    for (const stage of schedule.stages) {
        for (let i = 0; i < stage.repeat; i++) {
            total += stage.amount;
            months.push({ month: monthAt(monthIndex(schedule.startMonth) + months.length), planned: stage.amount, cumulativePlanned: total, actual: null, cumulativeActual: null, difference: null });
        }
    }
    return months;
}

export function lastMonthDate(month: string): string {
    const day = new Date(Date.UTC(Number(month.slice(0, 4)), Number(month.slice(5)), 0)).getUTCDate();
    return `${month}-${String(day).padStart(2, '0')}`;
}
export interface PlanEvent { date: string; amount: number; goalId?: string }
export function planEvents(schedule: MonthlyAssetPlanSchedule): PlanEvent[] {
    const events: PlanEvent[] = [];
    for (const stage of schedule.stages) {
        let count = 0;
        for (let index = monthIndex(stage.startDate?.slice(0, 7) || ''); index <= monthIndex(schedule.endMonth); index++) {
            const month = monthAt(index);
            const day = Math.min(stage.depositDay || 1, Number(lastMonthDate(month).slice(8)));
            const date = `${month}-${String(day).padStart(2, '0')}`;
            if (date < (stage.startDate || '')) continue;
            if (stage.endDate ? date > stage.endDate : count >= stage.repeat) break;
            events.push({ date, amount: stage.amount, goalId: stage.goalId });
            count++;
        }
    }
    for (const expense of schedule.expenses || []) events.push({ date: expense.date, amount: -expense.amount, goalId: expense.goalId });
    return events;
}
function validateGoalPlan(s: MonthlyAssetPlanSchedule): string | null {
    const count = monthIndex(s.endMonth) - monthIndex(s.startMonth) + 1;
    if (!Number.isFinite(count) || count < 1 || count > 120) return 'Choose a range of 1 to 120 months';
    const dateValid = (date: string): boolean => /^\d{4}-\d{2}-\d{2}$/.test(date) && Number(date.slice(8)) >= 1 && date <= lastMonthDate(date.slice(0, 7)) && Number.isFinite(monthIndex(date.slice(0, 7))) && date >= `${s.startMonth}-01` && date <= lastMonthDate(s.endMonth);
    const amountValid = (amount: number): boolean => Number.isSafeInteger(amount) && amount > 0;
    const nameValid = (name: string): boolean => !!name.trim() && new TextEncoder().encode(name).length <= 240;
    if (!s.finalGoal || !amountValid(s.finalGoal.amount) || s.finalGoal.date !== lastMonthDate(s.endMonth)) return 'Set a positive final net worth target';
    if (s.stages.length > 120 || (s.goals?.length || 0) > 120 || (s.expenses?.length || 0) > 120) return 'Use at most 120 items per section';
    const ids = new Set<string>();
    for (const goal of s.goals || []) {
        if (!goal.id || ids.has(goal.id) || !nameValid(goal.name) || !amountValid(goal.amount) || !dateValid(goal.date)) return 'Check goal names, amounts and dates';
        ids.add(goal.id);
    }
    for (const stage of s.stages) {
        if (!nameValid(stage.name || '') || !Number.isSafeInteger(stage.amount) || stage.amount < 0 || !dateValid(stage.startDate || '') || !Number.isInteger(stage.depositDay) || stage.depositDay! < 1 || stage.depositDay! > 31 || (stage.goalId && !ids.has(stage.goalId))) return 'Check savings names, amounts and dates';
        if (stage.endDate ? (!dateValid(stage.endDate) || stage.endDate < stage.startDate! || stage.repeat !== 0) : (!Number.isInteger(stage.repeat) || stage.repeat < 1 || stage.repeat > 120)) return 'Check savings duration';
        const deposits = planEvents({ ...s, stages: [stage], expenses: [] });
        if (!deposits.length || (!stage.endDate && deposits.length !== stage.repeat)) return 'Savings must fit within the plan dates';
    }
    for (const expense of s.expenses || []) {
        if (!nameValid(expense.name) || !amountValid(expense.amount) || !dateValid(expense.date) || (expense.goalId && !ids.has(expense.goalId))) return 'Check expense names, amounts and dates';
    }
    const events = planEvents(s);
    const income = events.filter(e => e.amount >= 0).reduce((sum, e) => sum + e.amount, 0);
    const expense = events.filter(e => e.amount < 0).reduce((sum, e) => sum - e.amount, 0);
    if (!Number.isSafeInteger(income) || !Number.isSafeInteger(expense)) return 'The total planned savings must be positive and within the amount limit';
    return null;
}
function previewGoalPlan(schedule: MonthlyAssetPlanSchedule): AssetPlanMonth[] {
    const events = planEvents(schedule);
    const months: AssetPlanMonth[] = [];
    let total = 0;
    for (let index = monthIndex(schedule.startMonth); index <= monthIndex(schedule.endMonth); index++) {
        const month = monthAt(index);
        const current = events.filter(event => event.date.startsWith(month));
        const income = current.filter(e => e.amount >= 0).reduce((sum, e) => sum + e.amount, 0);
        const expense = current.filter(e => e.amount < 0).reduce((sum, e) => sum - e.amount, 0);
        total += income - expense;
        months.push({ month, income, expense, planned: income - expense, cumulativePlanned: total, actual: null, cumulativeActual: null, difference: null });
    }
    return months;
}
