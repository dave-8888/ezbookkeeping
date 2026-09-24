import { beforeEach, describe, expect, jest, test } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';
import { getSocialGiftToken, setSocialGiftToken, socialGiftPrivacyHeaders } from '@/lib/social_gift_privacy.ts';
import { useSocialGiftStore } from '@/stores/socialGift.ts';
import services from '@/lib/services.ts';

jest.mock('@/lib/services.ts', () => ({
    __esModule: true,
    default: { getSocialGiftContacts: jest.fn(), getSocialGiftSummary: jest.fn(), getSocialGiftRecords: jest.fn() }
}));

describe('social gift privacy cache isolation', () => {
    beforeEach(() => { setActivePinia(createPinia()); setSocialGiftToken(''); jest.resetAllMocks(); });

    test('clearing access removes the request credential', () => {
        setSocialGiftToken('visit-one');
        expect(socialGiftPrivacyHeaders().headers['X-Social-Gift-Token']).toBe('visit-one');
        setSocialGiftToken('');
        expect(getSocialGiftToken()).toBe('');
    });

    test('late contact response cannot repopulate a locked page', async () => {
        const store = useSocialGiftStore();
        let respond!: (value: unknown) => void;
        jest.mocked(services.getSocialGiftContacts).mockReturnValue(new Promise<unknown>(resolve => { respond = resolve; }) as ReturnType<typeof services.getSocialGiftContacts>);
        setSocialGiftToken('visit-one');
        const request = store.loadContacts();
        setSocialGiftToken('');
        store.reset();
        respond({ data: { success: true, result: { items: [{ id: '1', name: 'Private contact' }], totalCount: 1 } } });
        await expect(request).rejects.toMatchObject({ message: 'social gift privacy is locked' });
        expect(store.contacts).toEqual([]);
        expect(store.contactsTotalCount).toBe(0);
    });

    test('a response from the previous unlock cannot overwrite a new visit', async () => {
        const store = useSocialGiftStore();
        let respond!: (value: unknown) => void;
        jest.mocked(services.getSocialGiftSummary).mockReturnValue(new Promise<unknown>(resolve => { respond = resolve; }) as ReturnType<typeof services.getSocialGiftSummary>);
        setSocialGiftToken('visit-one');
        const request = store.loadSummary();
        setSocialGiftToken('visit-two');
        respond({ data: { success: true, result: { balances: [], receivableContactCount: 4 } } });
        await expect(request).rejects.toMatchObject({ message: 'social gift privacy is locked' });
        expect(store.summary).toBeNull();
    });
});
