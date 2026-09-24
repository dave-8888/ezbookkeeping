<template>
    <v-container fluid class="pa-0 social-gifts-page">
        <div class="d-flex align-center mb-4 flex-wrap ga-2">
            <div>
                <h2 class="text-h5">{{ tt('Social Gifts') }}</h2>
                <div class="text-body-2 text-medium-emphasis">{{ tt('Track gifts sent, received, and balances by contact') }}</div>
            </div>
            <v-spacer />
            <v-btn variant="text" :prepend-icon="mdiAccountPlusOutline" @click="beginCreateContact()">{{ tt('New Contact') }}</v-btn>
            <v-btn color="primary" :prepend-icon="mdiPlus" @click="beginCreate()">{{ tt('Record Social Gift') }}</v-btn>
            <v-btn variant="text" :icon="mdiRefresh" :loading="loading" :aria-label="tt('Refresh')" @click="refresh" />
        </div>

        <v-alert v-if="loadError" type="error" variant="tonal" class="mb-4">{{ tt(loadError) }}</v-alert>

        <div v-if="summary && (summary.receivableContactCount || summary.payableContactCount)" class="d-flex ga-2 mb-4 flex-wrap">
            <v-chip variant="tonal">{{ tt('Receivable Contacts') }}: {{ summary.receivableContactCount }}</v-chip>
            <v-chip variant="tonal">{{ tt('Payable Contacts') }}: {{ summary.payableContactCount }}</v-chip>
        </div>

        <v-row class="mb-2" v-if="summary?.balances.length">
            <v-col cols="12" sm="6" lg="4" v-for="balance in summary.balances" :key="balance.currency">
                <v-card variant="outlined">
                    <v-card-text>
                        <div class="d-flex align-center"><strong>{{ balance.currency }}</strong><v-spacer /><v-chip size="small" :color="statusColor(balance.status)">{{ statusName(balance.status) }}</v-chip></div>
                        <div class="text-h6 mt-3">{{ formatAmount(Math.abs(balance.netAmount), balance.currency) }}</div>
                        <div class="text-caption text-medium-emphasis mt-1">{{ tt('Sent') }} {{ formatAmount(balance.sentAmount, balance.currency) }} · {{ tt('Received') }} {{ formatAmount(balance.receivedAmount, balance.currency) }}</div>
                    </v-card-text>
                </v-card>
            </v-col>
        </v-row>

        <v-card class="contact-card" variant="flat">
            <v-card-text class="contact-toolbar">
                <v-text-field v-model="keyword" density="compact" hide-details clearable :prepend-inner-icon="mdiMagnify" :label="tt('Search Contacts')" @keyup.enter="runSearch" @click:clear="clearSearch" />
                <v-btn-toggle v-model="status" mandatory class="status-filter" @update:model-value="runSearch"><v-btn v-for="option in statusOptions" :key="option.value" :value="option.value" size="small">{{ option.title }}</v-btn></v-btn-toggle>
            </v-card-text>
            <v-divider />
            <v-skeleton-loader v-if="loading" type="table-row@5" />
            <v-table v-else-if="contacts.length">
                <thead><tr><th>{{ tt('Contact') }}</th><th>{{ tt('Relationship') }}</th><th>{{ tt('Balance') }}</th><th>{{ tt('Last Activity') }}</th><th></th></tr></thead>
                <tbody>
                    <tr v-for="contact in contacts" :key="contact.id" class="cursor-pointer" @click="openContact(contact)">
                        <td><button class="contact-name" @click.stop="openContact(contact)">{{ contact.name }}</button><div class="text-caption text-medium-emphasis">{{ tt('records', { count: contact.recordCount }) }}</div></td>
                        <td>{{ contact.relationship || '-' }}</td>
                        <td>
                            <template v-if="contact.balances.length">
                                <div v-for="balance in contact.balances" :key="balance.currency" :class="`text-${statusColor(balance.status)}`">
                                    {{ statusName(balance.status) }} {{ formatAmount(Math.abs(balance.netAmount), balance.currency) }}
                                </div>
                            </template>
                            <span v-else>-</span>
                        </td>
                        <td>{{ contact.lastRecordTime ? formatRecordTime(contact.lastRecordTime) : '-' }}</td>
                        <td class="text-end"><v-icon :icon="mdiChevronRight" /></td>
                    </tr>
                </tbody>
            </v-table>
            <v-card-actions v-if="contactPageCount > 1"><v-spacer /><v-pagination v-model="contactPage" :length="contactPageCount" density="comfortable" @update:model-value="changeContactPage" /><v-spacer /></v-card-actions>
            <v-card-text v-if="!loading && !contacts.length" class="text-center py-12 text-medium-emphasis">
                <v-icon :icon="mdiGiftOutline" size="52" class="mb-3" />
                <div class="text-h6 mb-2">{{ tt(keyword || status !== 'all' ? 'No results' : 'No social gift contacts yet') }}</div><p class="mb-5">{{ tt(keyword || status !== 'all' ? 'socialGift.emptySearch' : 'socialGift.emptyHint') }}</p><v-btn v-if="keyword || status !== 'all'" variant="tonal" @click="resetSearch">{{ tt('Reset') }}</v-btn><v-btn v-else color="primary" variant="tonal" :prepend-icon="mdiPlus" @click="beginCreate()">{{ tt('Record Social Gift') }}</v-btn>
            </v-card-text>
        </v-card>

        <v-dialog v-model="showContactDialog" width="820" scrollable>
            <v-card v-if="selectedContact">
                <v-card-title class="d-flex align-center">
                    <div><div>{{ selectedContact.name }}</div><div class="text-caption text-medium-emphasis">{{ selectedContact.relationship }}<span v-if="selectedContact.comment"> · {{ selectedContact.comment }}</span></div></div>
                    <v-spacer />
                    <v-btn variant="text" :icon="mdiPencilOutline" @click="beginModifyContact(selectedContact)" />
                    <v-btn v-if="selectedContact.recordCount === 0" color="error" variant="text" :icon="mdiDeleteOutline" @click="confirmRemoveContact(selectedContact)" />
                    <v-btn color="primary" variant="text" :prepend-icon="mdiPlus" @click="beginCreate(selectedContact)">{{ tt('Add Record') }}</v-btn>
                </v-card-title>
                <v-divider />
                <v-card-text>
                    <v-list v-if="records.length" lines="three">
                        <template v-for="record in records" :key="record.id">
                            <v-list-item>
                                <template #prepend><v-avatar :color="record.direction === 'sent' ? 'warning' : 'success'" variant="tonal"><v-icon :icon="record.direction === 'sent' ? mdiArrowUp : mdiArrowDown" /></v-avatar></template>
                                <v-list-item-title>{{ directionName(record.direction) }} · {{ displayEventName(record.eventType, record.eventName) }}</v-list-item-title>
                                <v-list-item-subtitle>{{ [formatRecordTime(record.transaction.time), record.transaction.sourceAccount?.name, record.transaction.category?.name].filter(Boolean).join(' · ') }}</v-list-item-subtitle>
                                <template #append>
                                    <div class="text-end me-2"><strong>{{ formatAmount(record.transaction.sourceAmount, record.currency) }}</strong><div class="text-caption">{{ record.transaction.comment }}</div></div>
                                    <v-btn :icon="mdiPencilOutline" variant="text" size="small" @click="beginEdit(record)" />
                                    <v-btn :icon="mdiDeleteOutline" color="error" variant="text" size="small" @click="confirmRemove(record)" />
                                </template>
                            </v-list-item><v-divider />
                        </template>
                    </v-list>
                    <div v-else class="text-center text-medium-emphasis py-8">{{ tt('No social gift records yet') }}</div>
                    <v-pagination v-if="recordPageCount > 1" v-model="recordPage" :length="recordPageCount" density="comfortable" @update:model-value="changeRecordPage" />
                </v-card-text>
                <v-card-actions><v-spacer /><v-btn @click="showContactDialog = false">{{ tt('Close') }}</v-btn></v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="contactEditorOpen" width="520" persistent>
            <v-card>
                <v-card-title>{{ editingContact ? tt('Edit Contact') : tt('New Contact') }}</v-card-title>
                <v-card-text>
                    <v-text-field v-model="contactEditorName" :label="tt('Contact Name')" maxlength="64" />
                    <v-text-field v-model="contactEditorRelationship" :label="tt('Relationship')" maxlength="64" />
                    <v-textarea v-model="contactEditorComment" :label="tt('Contact Note')" maxlength="255" rows="2" />
                </v-card-text>
                <v-card-actions><v-spacer /><v-btn :disabled="saving" @click="contactEditorOpen = false">{{ tt('Cancel') }}</v-btn><v-btn color="primary" :loading="saving" :disabled="!contactEditorName.trim()" @click="submitContact">{{ tt('Save') }}</v-btn></v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="editorOpen" width="600" persistent scrollable>
            <v-card class="gift-editor">
                <v-card-title class="editor-heading">{{ editingRecord ? tt('Edit Social Gift') : tt('Record Social Gift') }}</v-card-title>
                <v-card-text class="editor-body">
                    <v-defaults-provider :defaults="{ VTextField: { variant: 'outlined', hideDetails: 'auto' }, VSelect: { variant: 'outlined', hideDetails: 'auto' }, VAutocomplete: { variant: 'outlined', hideDetails: 'auto' }, VTextarea: { variant: 'outlined', hideDetails: 'auto' } }">
                        <fieldset :disabled="saving" class="editor-fields">
                            <v-btn-toggle class="direction-control" mandatory v-model="direction" :disabled="saving" @update:model-value="changeDirection">
                                <v-btn value="sent" :prepend-icon="mdiArrowUp">{{ tt('Sent') }}</v-btn>
                                <v-btn value="received" :prepend-icon="mdiArrowDown">{{ tt('Received') }}</v-btn>
                            </v-btn-toggle>
                            <div class="amount-hero"><amount-input :currency="selectedCurrency" :show-currency="true" :label="tt('Amount')" v-model="amount" variant="outlined" :disabled="saving" /></div>
                            <div class="contact-label"><span>{{ tt('Contact') }}</span><v-btn v-if="!editingRecord" size="small" variant="text" color="primary" :disabled="saving" @click="useNewContact = !useNewContact">{{ tt(useNewContact ? 'socialGift.chooseExisting' : 'New Contact') }}</v-btn></div>
                            <v-text-field v-if="useNewContact" v-model="contactName" :label="tt('Contact Name')" maxlength="64" />
                            <v-autocomplete v-else v-model="contactId" :items="editorContacts" item-title="name" item-value="id" :label="tt('Contact')" :disabled="!!editingRecord || saving" />
                            <div class="editor-grid">
                                <v-select v-model="eventType" :items="eventOptions" item-title="title" item-value="value" :label="tt('Event Type')" />
                                <v-text-field type="datetime-local" v-model="transactionTime" :label="tt('Date and Time')" />
                            </div>
                            <v-text-field v-if="eventType === 'other'" v-model="eventName" :label="tt('Event Name')" maxlength="64" />
                            <div class="editor-grid">
                                <v-select clearable v-model="accountId" :items="accounts" item-title="name" item-value="id" :label="tt('Account')" />
                                <v-select clearable v-model="categoryId" :items="categoryOptions" item-title="name" item-value="id" :label="tt('Category')" />
                            </div>
                            <p class="text-caption text-medium-emphasis mb-4">{{ tt('socialGift.accountingHint') }}</p>
                            <v-expansion-panels v-model="morePanel" variant="accordion" class="more-details">
                                <v-expansion-panel :title="tt('socialGift.moreDetails')">
                                    <v-expansion-panel-text>
                                        <div class="optional-fields">
                                            <v-text-field v-if="eventType !== 'other'" v-model="eventName" :label="tt('Event Name')" maxlength="64" />
                                            <template v-if="useNewContact">
                                                <v-text-field v-model="relationship" :label="tt('Relationship')" maxlength="64" />
                                                <v-text-field v-model="contactComment" :label="tt('Contact Note')" maxlength="255" />
                                            </template>
                                            <v-textarea v-model="transactionComment" rows="2" :label="tt('Transaction Note')" maxlength="255" />
                                        </div>
                                    </v-expansion-panel-text>
                                </v-expansion-panel>
                            </v-expansion-panels>
                            <v-alert v-if="attemptedSave && validationMessage" type="warning" density="compact" variant="tonal" class="mt-4" role="alert">{{ tt(validationMessage) }}</v-alert>
                        </fieldset>
                    </v-defaults-provider>
                </v-card-text>
                <v-divider />
                <v-card-actions class="editor-actions"><v-btn :disabled="saving" @click="editorOpen = false">{{ tt('Cancel') }}</v-btn><v-spacer /><v-btn color="primary" variant="flat" :loading="saving" @click="submit">{{ tt('Save') }}</v-btn></v-card-actions>
            </v-card>
        </v-dialog>
        <snack-bar ref="snackbar" />
    </v-container>
</template>

<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from 'vue';
import SnackBar from '@/components/desktop/SnackBar.vue';
import { useI18n } from '@/locales/helpers.ts';
import { useSocialGiftPageBase } from '@/views/base/socialgifts/SocialGiftPageBase.ts';
import type { SocialGiftContactInfoResponse, SocialGiftDirection, SocialGiftEventType, SocialGiftRecordInfoResponse, SocialGiftStatus } from '@/models/social_gift.ts';
import { mdiAccountPlusOutline, mdiArrowDown, mdiArrowUp, mdiChevronRight, mdiDeleteOutline, mdiGiftOutline, mdiMagnify, mdiPencilOutline, mdiPlus, mdiRefresh } from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;
const { tt } = useI18n();
const snackbar = useTemplateRef<SnackBarType>('snackbar');
const showContactDialog = ref(false);
const base = useSocialGiftPageBase();
const { loading, saving, deleting, loadError, keyword, status, contactPage, recordPage, selectedContact, editingRecord, editorOpen, contactEditorOpen, editingContact, contactEditorName, contactEditorRelationship, contactEditorComment, useNewContact, contactId, contactName, relationship, contactComment, direction, eventType, eventName, accountId, categoryId, amount, transactionTime, transactionComment, contacts, records, summary, accounts, defaultCurrency, categoryOptions, contactPageCount, recordPageCount, validationMessage, formatAmount, formatRecordTime, load, searchContacts, selectContact, changeContactPage, changeRecordPage, beginCreateContact, beginModifyContact, saveContact, removeContact, applyDirectionDefaults, beginCreate, beginEdit, save, removeRecord } = base;

const attemptedSave = ref(false);
const morePanel = ref<number>();
const editorContacts = computed(() => selectedContact.value && !contacts.value.some(item => item.id === selectedContact.value?.id) ? [selectedContact.value, ...contacts.value] : contacts.value);
watch(editorOpen, value => { if (value) { attemptedSave.value = false; morePanel.value = transactionComment.value || (eventName.value && eventType.value !== 'other') ? 0 : undefined; } });
function runSearch(): void { searchContacts().catch(error => snackbar.value?.showMessage(error?.message || error)); }
function clearSearch(): void { keyword.value = ''; runSearch(); }
function resetSearch(): void { status.value = 'all'; clearSearch(); }

const selectedCurrency = computed(() => accounts.value.find(item => item.id === accountId.value)?.currency || defaultCurrency.value);
const statusOptions = computed(() => [{ value: 'all', title: tt('All') }, { value: 'receivable', title: tt('Receivable') }, { value: 'payable', title: tt('Payable') }, { value: 'settled', title: tt('Settled') }]);
const eventOptions = computed(() => (['wedding', 'newborn', 'birthday', 'housewarming', 'education', 'funeral', 'other'] as SocialGiftEventType[]).map(value => ({ value, title: tt(`socialGift.event.${value}`) })));
function statusName(value: SocialGiftStatus): string { return tt(`socialGift.status.${value}`); }
function statusColor(value: SocialGiftStatus): string { return value === 'receivable' ? 'warning' : value === 'payable' ? 'error' : 'success'; }
function directionName(value: SocialGiftDirection): string { return value === 'sent' ? tt('Sent') : tt('Received'); }
function displayEventName(type: SocialGiftEventType, name: string): string { return name || tt(`socialGift.event.${type}`); }
function refresh(): void { load(true).catch(error => snackbar.value?.showMessage(error?.message || error)); }
function openContact(contact: SocialGiftContactInfoResponse): void { selectContact(contact).then(() => { showContactDialog.value = true; }).catch(error => snackbar.value?.showMessage(error?.message || error)); }
function changeDirection(value: SocialGiftDirection): void { if (!editingRecord.value) applyDirectionDefaults(value); else categoryId.value = ''; }
function submit(): void { if (saving.value) return; attemptedSave.value = true; if (validationMessage.value) return; save().then(() => snackbar.value?.showMessage('Social gift record has been saved')).catch(error => snackbar.value?.showMessage(error?.message || error)); }
function confirmRemove(record: SocialGiftRecordInfoResponse): void { if (deleting.value || !window.confirm(tt('Are you sure you want to delete this social gift record?'))) return; removeRecord(record).then(() => snackbar.value?.showMessage('Social gift record has been deleted')).catch(error => snackbar.value?.showMessage(error?.message || error)); }
function submitContact(): void { saveContact().then(() => snackbar.value?.showMessage('Social gift contact has been saved')).catch(error => snackbar.value?.showMessage(error?.message || error)); }
function confirmRemoveContact(contact: SocialGiftContactInfoResponse): void { if (deleting.value || !window.confirm(tt('Are you sure you want to delete this contact?'))) return; removeContact(contact).then(() => { showContactDialog.value = false; snackbar.value?.showMessage('Social gift contact has been deleted'); }).catch(error => snackbar.value?.showMessage(error?.message || error)); }
load().catch(() => undefined);
</script>

<style scoped>
.social-gifts-page { max-width: 1440px; margin-inline: auto; }
.contact-card { border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity)); border-radius: 16px; }
.contact-toolbar { display: flex; align-items: center; gap: 24px; padding: 20px 24px; }
.contact-toolbar > :first-child { max-width: 380px; min-width: 180px; }
.status-filter { margin-left: auto; height: 40px !important; padding: 3px !important; background: rgba(var(--v-theme-on-surface), .04); border-radius: 9px; }
.status-filter :deep(.v-btn) { height: 34px !important; padding-inline: 14px !important; }
.status-filter :deep(.v-btn--active) { color: rgb(var(--v-theme-primary)); }
.contact-name { color: inherit; font-weight: 600; text-align: start; padding: 8px 0; }
.contact-name:focus-visible { outline: 2px solid rgb(var(--v-theme-primary)); outline-offset: 3px; }
.contact-card :deep(tbody tr) { height: 76px; }
.contact-card :deep(tbody tr:hover) { background: rgba(var(--v-theme-on-surface), .025); }
.gift-editor.v-card { border-radius: 20px !important; background: rgb(var(--v-theme-surface)) !important; }
.editor-heading { padding: 24px 28px 8px !important; font-size: 22px; font-weight: 600; }
.editor-body { padding: 16px 28px 24px !important; }
.editor-fields { border: 0; min-width: 0; padding: 0; }
.direction-control { width: 100%; height: 44px !important; padding: 4px !important; border-radius: 11px; background: rgba(var(--v-theme-on-surface), .05); }
.direction-control :deep(.v-btn) { width: 50% !important; flex: 1 1 50%; height: 36px !important; min-width: 0 !important; border-radius: 8px !important; background: transparent; }
.direction-control :deep(.v-btn--active) { background: rgb(var(--v-theme-surface)); box-shadow: 0 1px 5px #00000014; color: rgb(var(--v-theme-primary)); }
.direction-control :deep(.v-btn__overlay) { opacity: 0; }
.amount-hero { margin: 24px 0 12px; }
.amount-hero :deep(input) { font-size: 32px; font-weight: 500; min-height: 72px; font-variant-numeric: tabular-nums; }
.contact-label { display: flex; align-items: center; justify-content: space-between; min-height: 36px; margin-bottom: 8px; font-size: 13px; color: rgba(var(--v-theme-on-surface), .65); }
.editor-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 16px; margin: 20px 0; }
.more-details { border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity)); }
.more-details :deep(.v-expansion-panel) { background: transparent; }
.more-details :deep(.v-expansion-panel-title) { padding-inline: 0; font-size: 14px; color: rgba(var(--v-theme-on-surface), .65); }
.more-details :deep(.v-expansion-panel-text__wrapper) { padding: 8px 0 0; }
.optional-fields { display: grid; gap: 18px; }
.editor-actions { padding: 16px 28px !important; }
.editor-actions :deep(.v-btn) { min-width: 88px; border-radius: 10px; }
@media (max-width: 640px) {
    .contact-toolbar { flex-direction: column; align-items: stretch; gap: 16px; }
    .contact-toolbar > :first-child { max-width: none; }
    .status-filter { margin-left: 0; align-self: start; }
    .editor-heading { padding-inline: 20px !important; }
    .editor-body { padding-inline: 20px !important; }
    .editor-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
