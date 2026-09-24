import type { SocialGiftEventType, SocialGiftStatus } from '@/models/social_gift.ts';

export function getSocialGiftStatus(netAmount: number): SocialGiftStatus {
    if (netAmount > 0) return 'receivable';
    if (netAmount < 0) return 'payable';
    return 'settled';
}

export function isSocialGiftEventNameValid(eventType: SocialGiftEventType, eventName: string): boolean {
    return eventType !== 'other' || !!eventName.trim();
}

export function resolveValidSocialGiftSelection(preferredId: string | undefined, validIds: string[], fallbackId = ''): { value: string, preferenceValid: boolean } {
    if (preferredId && validIds.includes(preferredId)) return { value: preferredId, preferenceValid: true };
    return { value: validIds.includes(fallbackId) ? fallbackId : (fallbackId ? validIds[0] || '' : ''), preferenceValid: !preferredId };
}
