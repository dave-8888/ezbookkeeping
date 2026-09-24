<template>
    <div> 
        <f7-block><f7-button fill @click="beginCreate()">{{ tt('Record Social Gift') }}</f7-button></f7-block>

        <f7-block strong inset v-if="loadError"><div class="text-color-red">{{ tt(loadError) }}</div><f7-button outline class="margin-top" @click="load()">{{ tt('Retry') }}</f7-button></f7-block>
        <template v-else>
            <f7-block v-if="summary" class="display-flex justify-content-space-around"><f7-chip :text="`${tt('Receivable Contacts')}: ${summary.receivableContactCount}`" color="orange" /><f7-chip :text="`${tt('Payable Contacts')}: ${summary.payableContactCount}`" color="red" /></f7-block>
            <f7-card v-for="balance in summary?.balances || []" :key="balance.currency">
                <f7-card-header><strong>{{ balance.currency }}</strong><f7-chip :text="statusName(balance.status)" :color="statusColor(balance.status)" /></f7-card-header>
                <f7-card-content><div class="social-gift-total">{{ formatAmount(Math.abs(balance.netAmount), balance.currency) }}</div><small class="text-color-gray">{{ tt('Sent') }} {{ formatAmount(balance.sentAmount, balance.currency) }} · {{ tt('Received') }} {{ formatAmount(balance.receivedAmount, balance.currency) }}</small></f7-card-content>
            </f7-card>

            <f7-list strong inset dividers>
                <f7-list-input type="search" :placeholder="tt('Search Contacts')" v-model:value="keyword" @keyup.enter="searchContacts" />
                <f7-list-input type="select" :label="tt('Status')" v-model:value="status" @change="searchContacts">
                    <option value="all">{{ tt('All') }}</option><option value="receivable">{{ tt('Receivable') }}</option><option value="payable">{{ tt('Payable') }}</option><option value="settled">{{ tt('Settled') }}</option>
                </f7-list-input>
            </f7-list>
            <f7-block><f7-button outline @click="beginCreateContact()">{{ tt('New Contact') }}</f7-button></f7-block>
            <f7-block v-if="contactPageCount > 1" class="display-flex justify-content-space-between"><f7-button :disabled="contactPage <= 1" @click="changeContactPage(contactPage - 1)">{{ tt('Previous') }}</f7-button><span>{{ contactPage }} / {{ contactPageCount }}</span><f7-button :disabled="contactPage >= contactPageCount" @click="changeContactPage(contactPage + 1)">{{ tt('Next') }}</f7-button></f7-block>

            <f7-list strong inset dividers class="skeleton-text" v-if="loading"><f7-list-item title="Contact" after="0.00 CNY" v-for="i in 4" :key="i" /></f7-list>
            <f7-list strong inset dividers v-else-if="contacts.length">
                <f7-list-item v-for="contact in contacts" :key="contact.id" link="#" @click="openContact(contact)">
                    <template #title><strong>{{ contact.name }}</strong></template>
                    <template #footer>{{ contact.relationship || tt('No relationship') }} · {{ tt('records', { count: contact.recordCount }) }}</template>
                    <template #after><div class="text-align-right"><div v-for="balance in contact.balances" :key="balance.currency" :class="`text-color-${statusColor(balance.status)}`">{{ formatAmount(Math.abs(balance.netAmount), balance.currency) }}</div></div></template>
                </f7-list-item>
            </f7-list>
            <f7-block strong inset class="text-align-center text-color-gray" v-if="!loading && !contacts.length">{{ tt('No social gift contacts yet') }}</f7-block>
        </template>

        <f7-popup v-model:opened="showContactPopup">
            <f7-page>
                <f7-navbar :title="selectedContact?.name || tt('Social Gifts')"><f7-nav-left><f7-link :text="tt('Close')" @click="showContactPopup = false" /></f7-nav-left><f7-nav-right><f7-link v-if="selectedContact" icon-f7="pencil" @click="beginModifyContact(selectedContact)" /><f7-link icon-f7="plus" @click="selectedContact && beginCreate(selectedContact)" /></f7-nav-right></f7-navbar>
                <f7-block v-if="selectedContact?.relationship || selectedContact?.comment" class="text-color-gray">{{ selectedContact.relationship }}<span v-if="selectedContact.comment"> · {{ selectedContact.comment }}</span></f7-block>
                <f7-list strong inset dividers v-if="records.length">
                    <f7-list-item swipeout v-for="record in records" :key="record.id" link="#" @click="beginEdit(record)">
                        <template #media><f7-icon :f7="record.direction === 'sent' ? 'arrow_up' : 'arrow_down'" :color="record.direction === 'sent' ? 'orange' : 'green'" /></template>
                        <template #title>{{ directionName(record.direction) }} · {{ displayEventName(record.eventType, record.eventName) }}</template>
                        <template #footer>{{ formatRecordTime(record.transaction.time) }}<span v-if="record.transaction.comment"> · {{ record.transaction.comment }}</span></template>
                        <template #after><strong>{{ formatAmount(record.transaction.sourceAmount, record.currency) }}</strong></template>
                        <f7-swipeout-actions right><f7-swipeout-button color="red" close @click.stop="confirmRemove(record)">{{ tt('Delete') }}</f7-swipeout-button></f7-swipeout-actions>
                    </f7-list-item>
                </f7-list>
                <f7-block v-if="recordPageCount > 1" class="display-flex justify-content-space-between"><f7-button :disabled="recordPage <= 1" @click="changeRecordPage(recordPage - 1)">{{ tt('Previous') }}</f7-button><span>{{ recordPage }} / {{ recordPageCount }}</span><f7-button :disabled="recordPage >= recordPageCount" @click="changeRecordPage(recordPage + 1)">{{ tt('Next') }}</f7-button></f7-block>
                <f7-block strong inset class="text-align-center text-color-gray" v-if="!records.length">{{ tt('No social gift records yet') }}</f7-block>
            </f7-page>
        </f7-popup>

        <f7-popup v-model:opened="contactEditorOpen">
            <f7-page>
                <f7-navbar :title="editingContact ? tt('Edit Contact') : tt('New Contact')"><f7-nav-left><f7-link :text="tt('Cancel')" @click="contactEditorOpen = false" /></f7-nav-left><f7-nav-right><f7-link :text="tt('Save')" :class="{ disabled: saving || !contactEditorName.trim() }" @click="submitContact" /></f7-nav-right></f7-navbar>
                <f7-list strong inset dividers><f7-list-input :label="tt('Contact Name')" type="text" maxlength="64" v-model:value="contactEditorName" /><f7-list-input :label="tt('Relationship')" type="text" maxlength="64" v-model:value="contactEditorRelationship" /><f7-list-input :label="tt('Contact Note')" type="textarea" maxlength="255" v-model:value="contactEditorComment" /></f7-list>
                <f7-block v-if="editingContact?.recordCount === 0"><f7-button outline color="red" @click="confirmRemoveContact(editingContact)">{{ tt('Delete Contact') }}</f7-button></f7-block>
            </f7-page>
        </f7-popup>

        <f7-popup v-model:opened="editorOpen">
            <f7-page>
                <f7-navbar :title="editingRecord ? tt('Edit Social Gift') : tt('Record Social Gift')"><f7-nav-left><f7-link :text="tt('Cancel')" :class="{ disabled: saving }" @click="editorOpen = false" /></f7-nav-left><f7-nav-right><f7-link :text="tt('Save')" :class="{ disabled: saving || !!validationMessage }" @click="submit" /></f7-nav-right></f7-navbar>
                <f7-block><f7-segmented strong><f7-button :active="direction === 'sent'" @click="changeDirection('sent')">{{ tt('Sent') }}</f7-button><f7-button :active="direction === 'received'" @click="changeDirection('received')">{{ tt('Received') }}</f7-button></f7-segmented></f7-block>
                <f7-list strong inset dividers>
                    <f7-list-item v-if="!editingRecord" :title="tt('Create New Contact')"><template #after><f7-toggle :checked="useNewContact" @toggle:change="useNewContact = $event" /></template></f7-list-item>
                    <template v-if="useNewContact"><f7-list-input :label="tt('Contact Name')" type="text" maxlength="64" v-model:value="contactName" /><f7-list-input :label="tt('Relationship')" type="text" maxlength="64" v-model:value="relationship" /><f7-list-input :label="tt('Contact Note')" type="text" maxlength="255" v-model:value="contactComment" /></template>
                    <f7-list-input v-else type="select" :label="tt('Contact')" v-model:value="contactId"><option value="">{{ tt('Please select') }}</option><option v-for="contact in contacts" :key="contact.id" :value="contact.id">{{ contact.name }}{{ contact.relationship ? ` (${contact.relationship})` : '' }}</option></f7-list-input>
                    <f7-list-input type="select" :label="tt('Event Type')" v-model:value="eventType"><option v-for="option in eventOptions" :key="option.value" :value="option.value">{{ option.title }}</option></f7-list-input>
                    <f7-list-input :label="tt('Event Name')" type="text" maxlength="64" v-model:value="eventName" />
                    <f7-list-item link="#" no-chevron :header="tt('Amount')" :title="formatAmount(amount, selectedCurrency)" @click="showAmountInput = true" />
                    <f7-list-input type="datetime-local" :label="tt('Date and Time')" v-model:value="transactionTime" />
                    <f7-list-input type="select" :label="tt('Account')" v-model:value="accountId"><option value="">{{ tt('Please select') }}</option><option v-for="account in accounts" :key="account.id" :value="account.id">{{ account.name }} ({{ account.currency }})</option></f7-list-input>
                    <f7-list-input type="select" :label="tt('Transaction Category')" v-model:value="categoryId"><option value="">{{ tt('Please select') }}</option><option v-for="category in categoryOptions" :key="category.id" :value="category.id">{{ category.name }}</option></f7-list-input>
                    <f7-list-input type="textarea" :label="tt('Transaction Note')" maxlength="255" v-model:value="transactionComment" />
                </f7-list>
                <f7-block-footer>{{ tt('socialGift.accountingHint') }}</f7-block-footer>
                <f7-block strong inset class="text-color-orange" v-if="validationMessage">{{ tt(validationMessage) }}</f7-block>
                <number-pad-sheet :currency="selectedCurrency" v-model:show="showAmountInput" v-model="amount" />
            </f7-page>
        </f7-popup>
    </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { Router } from 'framework7/types';
import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';
import { useSocialGiftPageBase } from '@/views/base/socialgifts/SocialGiftPageBase.ts';
import type { SocialGiftContactInfoResponse, SocialGiftDirection, SocialGiftEventType, SocialGiftRecordInfoResponse, SocialGiftStatus } from '@/models/social_gift.ts';

const { tt } = useI18n();
const props = defineProps<{ f7route: Router.Route }>();
const { showToast, showConfirm } = useI18nUIComponents();
const showContactPopup = ref(false);
const showAmountInput = ref(false);
const base = useSocialGiftPageBase();
const { loading, saving, loadError, keyword, status, contactPage, recordPage, selectedContact, editingRecord, editorOpen, contactEditorOpen, editingContact, contactEditorName, contactEditorRelationship, contactEditorComment, useNewContact, contactId, contactName, relationship, contactComment, direction, eventType, eventName, accountId, categoryId, amount, transactionTime, transactionComment, contacts, records, summary, accounts, defaultCurrency, categoryOptions, contactPageCount, recordPageCount, validationMessage, formatAmount, formatRecordTime, load, searchContacts, selectContact, changeContactPage, changeRecordPage, beginCreateContact, beginModifyContact, saveContact, removeContact, applyDirectionDefaults, beginCreate, beginEdit, save, removeRecord } = base;
const selectedCurrency = computed(() => accounts.value.find(item => item.id === accountId.value)?.currency || defaultCurrency.value);
const eventOptions = computed(() => (['wedding', 'newborn', 'birthday', 'housewarming', 'education', 'funeral', 'other'] as SocialGiftEventType[]).map(value => ({ value, title: tt(`socialGift.event.${value}`) })));
function statusName(value: SocialGiftStatus): string { return tt(`socialGift.status.${value}`); }
function statusColor(value: SocialGiftStatus): string { return value === 'receivable' ? 'orange' : value === 'payable' ? 'red' : 'green'; }
function directionName(value: SocialGiftDirection): string { return value === 'sent' ? tt('Sent') : tt('Received'); }
function displayEventName(type: SocialGiftEventType, name: string): string { return name || tt(`socialGift.event.${type}`); }

defineExpose({ refresh: () => load(true) });
function openContact(contact: SocialGiftContactInfoResponse): void { selectContact(contact).then(() => { showContactPopup.value = true; }).catch(error => showToast(error?.message || error)); }
function changeDirection(value: SocialGiftDirection): void { if (!editingRecord.value) applyDirectionDefaults(value); else { direction.value = value; categoryId.value = ''; } }
function submit(): void { save().then(() => showToast('Social gift record has been saved')).catch(error => showToast(error?.message || error)); }
function confirmRemove(record: SocialGiftRecordInfoResponse): void { showConfirm('Are you sure you want to delete this social gift record?', () => removeRecord(record).then(() => showToast('Social gift record has been deleted')).catch(error => showToast(error?.message || error))); }
function submitContact(): void { saveContact().then(() => showToast('Social gift contact has been saved')).catch(error => showToast(error?.message || error)); }
function confirmRemoveContact(contact: SocialGiftContactInfoResponse): void { showConfirm('Are you sure you want to delete this contact?', () => removeContact(contact).then(() => { showContactPopup.value = false; showToast('Social gift contact has been deleted'); }).catch(error => showToast(error?.message || error))); }
load().then(() => {
    if (props.f7route.query['new'] === '1') beginCreate();
}).catch(() => undefined);
</script>

<style scoped>.social-gift-total { font-size: 1.5rem; font-weight: 600; }</style>
