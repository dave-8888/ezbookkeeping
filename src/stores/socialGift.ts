import { getSocialGiftToken } from '@/lib/social_gift_privacy.ts';
import { ref } from 'vue';
import { defineStore } from 'pinia';

import services from '@/lib/services.ts';
import type {
    SocialGiftContactInfoResponse,
    SocialGiftRecordInfoResponse,
    SocialGiftSummaryResponse,
    SocialGiftRecordCreateRequest,
    SocialGiftRecordModifyRequest,
    SocialGiftContactModifyRequest,
    SocialGiftContactCreateRequest
} from '@/models/social_gift.ts';

function resultOrReject<T>(data: { success?: boolean, result?: T } | undefined, fallback: string): T | Promise<never> {
    if (!data?.success || data.result === undefined) {
        return Promise.reject({ message: fallback }) as Promise<never>;
    }
    return data.result;
}

export const useSocialGiftStore = defineStore('socialGift', () => {
    const contacts = ref<SocialGiftContactInfoResponse[]>([]);
    const contactsTotalCount = ref<number>(0);
    const records = ref<SocialGiftRecordInfoResponse[]>([]);
    const recordsTotalCount = ref<number>(0);
    const summary = ref<SocialGiftSummaryResponse | null>(null);

    function loadContacts(keyword = '', status = 'all', page = 1, count = 50): Promise<SocialGiftContactInfoResponse[]> {
        const accessToken = getSocialGiftToken();
        return services.getSocialGiftContacts({ keyword, status, page, count }).then(response => {
            if (!accessToken || accessToken !== getSocialGiftToken()) return Promise.reject({ message: 'social gift privacy is locked' });
            const result = resultOrReject(response.data, 'Unable to retrieve social gift contacts');
            if (result instanceof Promise) return result;
            contacts.value = result.items;
            contactsTotalCount.value = result.totalCount;
            return result.items;
        });
    }

    function loadSummary(): Promise<SocialGiftSummaryResponse> {
        const accessToken = getSocialGiftToken();
        return services.getSocialGiftSummary().then(response => {
            if (!accessToken || accessToken !== getSocialGiftToken()) return Promise.reject({ message: 'social gift privacy is locked' });
            const result = resultOrReject(response.data, 'Unable to retrieve social gift summary');
            if (result instanceof Promise) return result;
            summary.value = result;
            return result;
        });
    }

    function loadContact(id: string): Promise<SocialGiftContactInfoResponse> {
        return services.getSocialGiftContact({ id }).then(response => resultOrReject(response.data, 'Unable to retrieve social gift contacts') as SocialGiftContactInfoResponse);
    }

    function loadRecords(contactId: string, page = 1, count = 50): Promise<SocialGiftRecordInfoResponse[]> {
        const accessToken = getSocialGiftToken();
        return services.getSocialGiftRecords({ contactId, page, count }).then(response => {
            if (!accessToken || accessToken !== getSocialGiftToken()) return Promise.reject({ message: 'social gift privacy is locked' });
            const result = resultOrReject(response.data, 'Unable to retrieve social gift records');
            if (result instanceof Promise) return result;
            records.value = result.items;
            recordsTotalCount.value = result.totalCount;
            return result.items;
        });
    }

    function saveRecord(req: SocialGiftRecordCreateRequest | SocialGiftRecordModifyRequest): Promise<SocialGiftRecordInfoResponse> {
        const promise = 'id' in req ? services.modifySocialGiftRecord(req) : services.createSocialGiftRecord(req);
        return promise.then(response => resultOrReject(response.data, 'Unable to save social gift record') as SocialGiftRecordInfoResponse);
    }

    function deleteRecord(id: string): Promise<boolean> {
        return services.deleteSocialGiftRecord({ id }).then(response => resultOrReject(response.data, 'Unable to delete social gift record') as boolean);
    }

    function modifyContact(req: SocialGiftContactModifyRequest): Promise<SocialGiftContactInfoResponse> {
        return services.modifySocialGiftContact(req).then(response => resultOrReject(response.data, 'Unable to save social gift contact') as SocialGiftContactInfoResponse);
    }

    function createContact(req: SocialGiftContactCreateRequest): Promise<SocialGiftContactInfoResponse> {
        return services.createSocialGiftContact(req).then(response => resultOrReject(response.data, 'Unable to save social gift contact') as SocialGiftContactInfoResponse);
    }

    function deleteContact(id: string): Promise<boolean> {
        return services.deleteSocialGiftContact({ id }).then(response => resultOrReject(response.data, 'Unable to delete social gift contact') as boolean);
    }

    function reset(): void {
        contacts.value = [];
        records.value = [];
        summary.value = null;
        contactsTotalCount.value = 0;
        recordsTotalCount.value = 0;
    }

    return { contacts, contactsTotalCount, records, recordsTotalCount, summary, loadContacts, loadContact, loadSummary, loadRecords, saveRecord, deleteRecord, createContact, modifyContact, deleteContact, reset };
});
