import { computed, ref } from 'vue';

import { CategoryType } from '@/core/category.ts';
import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';
import { useI18n } from '@/locales/helpers.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';
import type {
    SocialGiftContactInfoResponse,
    SocialGiftDirection,
    SocialGiftEventType,
    SocialGiftRecordCreateRequest,
    SocialGiftRecordInfoResponse,
    SocialGiftRecordModifyRequest
} from '@/models/social_gift.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useSocialGiftStore } from '@/stores/socialGift.ts';
import { useUserStore } from '@/stores/user.ts';
import { generateRandomUUID } from '@/lib/misc.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';
import { isSocialGiftEventNameValid } from '@/lib/social_gift.ts';

const pageSize = 20;

function localDateTimeValue(date: Date): string {
    const pad = (value: number) => value.toString().padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function useSocialGiftPageBase() {
    const { tt, formatAmountToLocalizedNumeralsWithCurrency, formatDateTimeToLongDateTime } = useI18n();
    const accountsStore = useAccountsStore();
    const categoriesStore = useTransactionCategoriesStore();
    const settingsStore = useSettingsStore();
    const socialGiftStore = useSocialGiftStore();
    const userStore = useUserStore();

    const loading = ref(true);
    const saving = ref(false);
    const deleting = ref(false);
    const loadError = ref('');
    const keyword = ref('');
    const status = ref('all');
    const contactPage = ref(1);
    const recordPage = ref(1);
    const selectedContact = ref<SocialGiftContactInfoResponse | null>(null);
    const editingRecord = ref<SocialGiftRecordInfoResponse | null>(null);
    const editorOpen = ref(false);
    const contactEditorOpen = ref(false);
    const editingContact = ref<SocialGiftContactInfoResponse | null>(null);
    const contactEditorName = ref('');
    const contactEditorRelationship = ref('');
    const contactEditorComment = ref('');
    const useNewContact = ref(false);
    const contactId = ref('');
    const contactName = ref('');
    const relationship = ref('');
    const contactComment = ref('');
    const direction = ref<SocialGiftDirection>('sent');
    const eventType = ref<SocialGiftEventType>('wedding');
    const eventName = ref('');
    const accountId = ref('');
    const categoryId = ref('');
    const amount = ref(0);
    const transactionTime = ref(localDateTimeValue(new Date()));
    const transactionComment = ref('');

    const contacts = computed(() => socialGiftStore.contacts);
    const records = computed(() => socialGiftStore.records);
    const summary = computed(() => socialGiftStore.summary);
    const contactPageCount = computed(() => Math.max(1, Math.ceil(socialGiftStore.contactsTotalCount / pageSize)));
    const recordPageCount = computed(() => Math.max(1, Math.ceil(socialGiftStore.recordsTotalCount / pageSize)));
    const accounts = computed(() => accountsStore.allVisiblePlainAccounts);
    const categoryOptions = computed<TransactionCategory[]>(() => {
        const type = direction.value === 'sent' ? CategoryType.Expense : CategoryType.Income;
        const result: TransactionCategory[] = [];
        for (const parent of categoriesStore.allTransactionCategories[type] || []) {
            for (const child of parent.subCategories || []) {
                if (!child.hidden) result.push(child);
            }
        }
        return result;
    });
    const validationMessage = computed(() => {
        if (useNewContact.value && !contactName.value.trim()) return 'Contact name is required';
        if (!useNewContact.value && !contactId.value) return 'Please select a contact';
        if (!isSocialGiftEventNameValid(eventType.value, eventName.value)) return 'Event name is required';
        if (amount.value <= 0) return 'Amount must be greater than zero';
        if (!transactionTime.value) return 'Please select a date and time';
        return '';
    });

    function formatAmount(amountValue: number, currency: string): string {
        const visible = settingsStore.appSettings.showAccountBalance;
        return formatAmountToLocalizedNumeralsWithCurrency(visible ? amountValue : DISPLAY_HIDDEN_AMOUNT, currency);
    }

    function formatRecordTime(timestamp: number): string {
        return formatDateTimeToLongDateTime(parseDateTimeFromUnixTime(timestamp));
    }

    function load(force = false): Promise<void> {
        loading.value = true;
        loadError.value = '';
        return Promise.all([
            accountsStore.loadAllAccounts({ force }),
            categoriesStore.loadAllCategories({ force }),
            socialGiftStore.loadContacts(keyword.value, status.value, contactPage.value, pageSize),
            socialGiftStore.loadSummary()
        ]).then(() => {
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            loadError.value = error?.message || String(error);
            return Promise.reject(error);
        });
    }

    function reloadList(): Promise<void> {
        const selectedId = selectedContact.value?.id;
        return Promise.all([
            socialGiftStore.loadContacts(keyword.value, status.value, contactPage.value, pageSize),
            socialGiftStore.loadSummary(),
            selectedId ? socialGiftStore.loadRecords(selectedId, recordPage.value, pageSize) : Promise.resolve([]),
            selectedId ? socialGiftStore.loadContact(selectedId) : Promise.resolve(null)
        ]).then(([, , , refreshedContact]) => {
            if (refreshedContact) selectedContact.value = refreshedContact;
        });
    }

    function selectContact(contact: SocialGiftContactInfoResponse): Promise<void> {
        selectedContact.value = contact;
        recordPage.value = 1;
        return socialGiftStore.loadRecords(contact.id, recordPage.value, pageSize).then(() => undefined);
    }

    function searchContacts(): Promise<void> {
        contactPage.value = 1;
        return reloadList();
    }

    function changeContactPage(page: number): Promise<void> {
        contactPage.value = page;
        return socialGiftStore.loadContacts(keyword.value, status.value, page, pageSize).then(() => undefined);
    }

    function changeRecordPage(page: number): Promise<void> {
        recordPage.value = page;
        if (!selectedContact.value) return Promise.resolve();
        return socialGiftStore.loadRecords(selectedContact.value.id, page, pageSize).then(() => undefined);
    }

    function beginCreateContact(): void {
        editingContact.value = null;
        contactEditorName.value = '';
        contactEditorRelationship.value = '';
        contactEditorComment.value = '';
        contactEditorOpen.value = true;
    }

    function beginModifyContact(contact: SocialGiftContactInfoResponse): void {
        editingContact.value = contact;
        contactEditorName.value = contact.name;
        contactEditorRelationship.value = contact.relationship;
        contactEditorComment.value = contact.comment;
        contactEditorOpen.value = true;
    }

    function saveContact(): Promise<SocialGiftContactInfoResponse> {
        if (!contactEditorName.value.trim()) return Promise.reject({ message: tt('Contact name is required') });
        saving.value = true;
        const common = { name: contactEditorName.value.trim(), relationship: contactEditorRelationship.value.trim(), comment: contactEditorComment.value.trim() };
        const promise = editingContact.value ? socialGiftStore.modifyContact({ ...common, id: editingContact.value.id }) : socialGiftStore.createContact(common);
        return promise.then(result => {
            if (selectedContact.value?.id === result.id) selectedContact.value = result;
            contactEditorOpen.value = false;
            return searchContacts().then(() => result);
        }).finally(() => { saving.value = false; });
    }

    function removeContact(contact: SocialGiftContactInfoResponse): Promise<void> {
        deleting.value = true;
        return socialGiftStore.deleteContact(contact.id).then(() => {
            if (selectedContact.value?.id === contact.id) selectedContact.value = null;
            contactEditorOpen.value = false;
            return searchContacts();
        }).finally(() => { deleting.value = false; });
    }

    function applyDirectionDefaults(nextDirection: SocialGiftDirection): void {
        direction.value = nextDirection;
        accountId.value = '';
        categoryId.value = '';
    }

    function beginCreate(contact?: SocialGiftContactInfoResponse): void {
        editingRecord.value = null;
        contactId.value = contact?.id || '';
        useNewContact.value = !contact && contacts.value.length === 0;
        contactName.value = '';
        relationship.value = '';
        contactComment.value = '';
        eventType.value = 'wedding';
        eventName.value = '';
        amount.value = 0;
        transactionTime.value = localDateTimeValue(new Date());
        transactionComment.value = '';
        applyDirectionDefaults('sent');
        editorOpen.value = true;
    }

    function beginEdit(record: SocialGiftRecordInfoResponse): void {
        editingRecord.value = record;
        useNewContact.value = false;
        contactId.value = record.contactId;
        direction.value = record.direction;
        eventType.value = record.eventType;
        eventName.value = record.eventName;
        accountId.value = record.transaction.sourceAccountId === '0' ? '' : record.transaction.sourceAccountId;
        categoryId.value = record.transaction.categoryId === '0' ? '' : record.transaction.categoryId;
        amount.value = record.transaction.sourceAmount;
        transactionTime.value = localDateTimeValue(new Date(record.transaction.time * 1000));
        transactionComment.value = record.transaction.comment;
        editorOpen.value = true;
    }

    function save(): Promise<SocialGiftRecordInfoResponse> {
        if (validationMessage.value) return Promise.reject({ message: tt(validationMessage.value) });
        saving.value = true;
        const date = new Date(transactionTime.value);
        const common = {
            contactId: useNewContact.value ? '0' : contactId.value,
            direction: direction.value,
            eventType: eventType.value,
            eventName: eventName.value.trim(),
            categoryId: categoryId.value || '0',
            time: Math.floor(date.getTime() / 1000),
            utcOffset: -date.getTimezoneOffset(),
            accountId: accountId.value || '0',
            currency: accounts.value.find(item => item.id === accountId.value)?.currency || editingRecord.value?.currency || userStore.currentUserDefaultCurrency,
            amount: amount.value,
            comment: transactionComment.value.trim()
        };
        let request: SocialGiftRecordCreateRequest | SocialGiftRecordModifyRequest;
        if (editingRecord.value) {
            request = { ...common, id: editingRecord.value.id };
        } else {
            request = {
                ...common,
                clientSessionId: generateRandomUUID(),
                newContact: useNewContact.value ? { name: contactName.value.trim(), relationship: relationship.value.trim(), comment: contactComment.value.trim() } : undefined
            };
        }
        return socialGiftStore.saveRecord(request).then(result => {
            editorOpen.value = false;
            return reloadList().then(() => result);
        }).finally(() => { saving.value = false; });
    }

    function removeRecord(record: SocialGiftRecordInfoResponse): Promise<void> {
        deleting.value = true;
        return socialGiftStore.deleteRecord(record.id).then(() => reloadList()).finally(() => { deleting.value = false; });
    }

    return {
        loading, saving, deleting, loadError, keyword, status, contactPage, recordPage, selectedContact, editingRecord, editorOpen,
        contactEditorOpen, editingContact, contactEditorName, contactEditorRelationship, contactEditorComment,
        useNewContact, contactId, contactName, relationship, contactComment, direction, eventType, eventName,
        accountId, categoryId, amount, transactionTime, transactionComment, contacts, records, summary, accounts,
        defaultCurrency: computed(() => editingRecord.value?.currency || userStore.currentUserDefaultCurrency), categoryOptions, contactPageCount, recordPageCount, validationMessage, formatAmount, formatRecordTime, load, reloadList, selectContact,
        searchContacts, changeContactPage, changeRecordPage, beginCreateContact, beginModifyContact, saveContact, removeContact,
        applyDirectionDefaults, beginCreate, beginEdit, save, removeRecord
    };
}
