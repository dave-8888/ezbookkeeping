import { onScopeDispose, watch } from 'vue';

// Coalesce bursts, serialize refreshes, and retain notifications received while
// another page load is in flight. No polling or work after the page is unmounted.
export function useCoalescedRefresh(revision: () => number, busy: () => boolean, refresh: () => Promise<void>): void {
    let pending = false;
    let running = false;
    let disposed = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    function schedule(): void {
        if (timer !== undefined) clearTimeout(timer);
        timer = undefined;
        if (disposed || !pending || running || busy()) return;
        timer = setTimeout(async () => {
            timer = undefined;
            if (disposed || busy()) return;
            pending = false;
            running = true;
            try {
                await refresh();
            } finally {
                running = false;
                schedule();
            }
        }, 400);
    }

    watch(revision, () => {
        pending = true;
        schedule();
    });
    watch(busy, schedule);
    onScopeDispose(() => {
        disposed = true;
        if (timer !== undefined) clearTimeout(timer);
    });
}
