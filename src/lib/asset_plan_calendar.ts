import type { MonthlyAssetPlanResponse } from '@/models/asset_plan.ts';
import { lastMonthDate, planEvents } from '@/lib/asset_plan_monthly.ts';

export interface CalendarEvent { date: string; name: string; amount: number; kind: 'income' | 'expense' }

// Use UTC for layout so DST and browser timezone never move a calendar date.
export function calendarDates(month: string): (string | null)[] {
    const first = new Date(`${month}-01T00:00:00Z`);
    const offset = (first.getUTCDay() + 6) % 7;
    const count = Number(lastMonthDate(month).slice(8));
    return Array.from({ length: Math.ceil((offset + count) / 7) * 7 }, (_, index) => {
        const day = index - offset + 1;
        return day < 1 || day > count ? null : `${month}-${String(day).padStart(2, '0')}`;
    });
}

export function calendarEvents(plan: MonthlyAssetPlanResponse): CalendarEvent[] {
    if (plan.version !== 2) return [];
    const events: CalendarEvent[] = [];
    for (const stage of plan.stages) {
        for (const event of planEvents({ ...plan, stages: [stage], expenses: [] })) {
            events.push({ date: event.date, name: stage.name || '', amount: event.amount, kind: 'income' });
        }
    }
    for (const expense of plan.expenses || []) {
        events.push({ ...expense, kind: 'expense' });
    }
    return events.filter(event => !plan.tracking || event.date > plan.baselineDate);
}
