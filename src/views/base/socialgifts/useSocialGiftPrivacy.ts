import { onBeforeUnmount, onDeactivated, ref } from 'vue';
import services from '@/lib/services.ts';
import { setSocialGiftToken } from '@/lib/social_gift_privacy.ts';
import { useSocialGiftStore } from '@/stores/socialGift.ts';

export function useSocialGiftPrivacy() {
    const unlocked = ref(false);
    const configured = ref<boolean | null>(null);
    const busy = ref(false);
    const error = ref('');
    const store = useSocialGiftStore();
    let generation = 0;

    function lock(): void {
        generation++;
        if (unlocked.value) void services.lockSocialGiftPrivacy().catch(() => undefined);
        unlocked.value = false;
        setSocialGiftToken('');
        store.reset();
    }
    async function status(): Promise<void> {
        error.value = '';
        try {
            const response = await services.getSocialGiftPrivacyStatus();
            if (!response.data.success || !response.data.result) throw new Error('Unable to load privacy settings');
            configured.value = response.data.result.configured;
        } catch { error.value = 'Unable to load privacy settings'; }
    }
    async function unlock(password: string): Promise<void> {
        if (busy.value || configured.value === null) return;
        busy.value = true;
        error.value = '';
        const current = generation;
        try {
            const response = await services.unlockSocialGiftPrivacy(password, !configured.value);
            if (!response.data.success || !response.data.result?.token) throw new Error('Unable to unlock social gifts');
            if (current !== generation) {
                void services.lockSocialGiftPrivacy(response.data.result.token).catch(() => undefined);
                return;
            }
            setSocialGiftToken(response.data.result.token);
            configured.value = true;
            unlocked.value = true;
        } catch (reason) {
            if (current === generation) {
                const failure = reason as { response?: { data?: { errorMessage?: string } } };
                error.value = failure.response?.data?.errorMessage || 'Unable to unlock social gifts';
            }
        } finally { busy.value = false; }
    }
    lock();
    void status();
    window.addEventListener('pagehide', lock);
    onDeactivated(lock);
    onBeforeUnmount(() => { lock(); window.removeEventListener('pagehide', lock); });
    return { unlocked, configured, busy, error, lock, unlock, status };
}
