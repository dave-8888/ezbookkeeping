import { afterEach, describe, expect, jest, test } from '@jest/globals';
import { effectScope, nextTick, ref } from 'vue';
import { useCoalescedRefresh } from '@/components/base/CoalescedRefresh.ts';

afterEach(() => { jest.useRealTimers(); });

describe('transaction list refresh scheduling', () => {
    test('coalesces bursts, waits for page loads, and cancels on unmount', async () => {
        jest.useFakeTimers();
        const revision = ref(0);
        const busy = ref(false);
        const refresh = jest.fn<() => Promise<void>>().mockResolvedValue();
        const scope = effectScope();
        scope.run(() => useCoalescedRefresh(() => revision.value, () => busy.value, refresh));
        revision.value++;
        await nextTick();
        await jest.advanceTimersByTimeAsync(200);
        revision.value++;
        await nextTick();
        await jest.advanceTimersByTimeAsync(399);
        expect(refresh).not.toHaveBeenCalled();
        await jest.advanceTimersByTimeAsync(1);
        expect(refresh).toHaveBeenCalledTimes(1);
        busy.value = true;
        revision.value++;
        await nextTick();
        await jest.advanceTimersByTimeAsync(1000);
        expect(refresh).toHaveBeenCalledTimes(1);
        busy.value = false;
        await nextTick();
        await jest.advanceTimersByTimeAsync(400);
        expect(refresh).toHaveBeenCalledTimes(2);
        revision.value++;
        await nextTick();
        scope.stop();
        await jest.advanceTimersByTimeAsync(400);
        expect(refresh).toHaveBeenCalledTimes(2);
    });

    test('retains changes during a refresh without overlapping requests', async () => {
        jest.useFakeTimers();
        const revision = ref(0);
        let finish!: () => void;
        const refresh = jest.fn<() => Promise<void>>()
            .mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }))
            .mockResolvedValue();
        const scope = effectScope();
        scope.run(() => useCoalescedRefresh(() => revision.value, () => false, refresh));
        revision.value++;
        await nextTick();
        await jest.advanceTimersByTimeAsync(400);
        revision.value++;
        await nextTick();
        await jest.advanceTimersByTimeAsync(1000);
        expect(refresh).toHaveBeenCalledTimes(1);
        finish();
        await jest.advanceTimersByTimeAsync(400);
        expect(refresh).toHaveBeenCalledTimes(2);
        scope.stop();
    });
});
