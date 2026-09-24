<template>
    <f7-page ptr @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-title :title="tt('global.app.title')"></f7-nav-title>
        </f7-navbar>

        <f7-card class="home-summary-card no-margin-top" :class="{ 'skeleton-text': loading }">
            <f7-card-header class="display-block home-summary-header">
                <f7-link class="home-summary-details" :href="`/transaction/list?${overviewStore.getTransactionListPageParams({ dateType: DateRange.ThisMonth.type })}`"
                         :aria-label="`${tt('This Month')} · ${tt('Details')}`">
                    <p class="no-margin">
                        <span class="card-header-content" v-if="loading">
                            <span class="home-summary-month">Month</span>
                            <span>·</span>
                            <small>Expense</small>
                        </span>
                        <span class="card-header-content" v-else-if="!loading">
                            <span class="home-summary-month">{{ displayDateRange?.thisMonth?.displayTime }}</span>
                            <span>·</span>
                            <small>{{ tt('Expense') }}</small>
                        </span>
                    </p>
                    <p class="no-margin">
                        <span class="month-expense" v-if="loading">0.00 USD</span>
                        <span class="month-expense" v-else-if="!loading">{{ transactionOverview && transactionOverview.thisMonth ? getDisplayExpenseAmount(transactionOverview.thisMonth) : '-' }}</span>
                    </p>
                    <p class="no-margin">
                        <small class="home-summary-misc" v-if="loading">Monthly income 0.00 USD</small>
                        <small class="home-summary-misc" v-else-if="!loading">
                            <span>{{ tt('Monthly income') }}</span>
                            <span>{{ transactionOverview && transactionOverview.thisMonth ? getDisplayIncomeAmount(transactionOverview.thisMonth) : '-' }}</span>
                        </small>
                    </p>
                </f7-link>
                <button type="button" class="home-amount-toggle"
                        :aria-label="tt(showAmountInHomePage ? 'Hide Amount' : 'Show Amount')"
                        @click="showAmountInHomePage = !showAmountInHomePage">
                    <f7-icon class="ebk-hide-icon" :f7="showAmountInHomePage ? 'eye_slash_fill' : 'eye_fill'"></f7-icon>
                </button>
            </f7-card-header>
        </f7-card>

        <f7-list strong inset dividers class="margin-top overview-transaction-list" :class="{ 'skeleton-text': loading }">
            <f7-list-item :link="`/transaction/list?${overviewStore.getTransactionListPageParams({ dateType: DateRange.Today.type })}`" chevron-center>
                <template #title>
                    <div class="padding-top-half">
                        <span v-if="loading">Today</span>
                        <span v-else-if="!loading">{{ tt('Today') }}</span>
                    </div>
                </template>
                <template #after>
                    <div class="overview-transaction-amount">
                        <div class="text-income text-align-right">
                            <small class="overview-amount-label">{{ tt('Income') }}</small>
                            <small v-if="loading">0.00 USD</small>
                            <small v-else-if="!loading && transactionOverview.today && transactionOverview.today.valid">{{ getDisplayIncomeAmount(transactionOverview.today) }}</small>
                        </div>
                        <div class="text-expense text-align-right">
                            <small class="overview-amount-label">{{ tt('Expense') }}</small>
                            <small v-if="loading">0.00 USD</small>
                            <small v-else-if="!loading && transactionOverview.today && transactionOverview.today.valid">{{ getDisplayExpenseAmount(transactionOverview.today) }}</small>
                        </div>
                    </div>
                </template>
            </f7-list-item>

            <f7-list-item :link="`/transaction/list?${overviewStore.getTransactionListPageParams({ dateType: DateRange.ThisWeek.type })}`" chevron-center>
                <template #title>
                    <div class="padding-top-half">
                        <span v-if="loading">This Week</span>
                        <span v-else-if="!loading">{{ tt('This Week') }}</span>
                    </div>
                </template>
                <template #footer>
                    <div class="overview-transaction-footer padding-bottom-half">
                        <span v-if="loading">MM/DD</span>
                        <span v-else-if="!loading">{{ displayDateRange?.thisWeek?.startTime }}</span>
                        <span>-</span>
                        <span v-if="loading">MM/DD</span>
                        <span v-else-if="!loading">{{ displayDateRange?.thisWeek?.endTime }}</span>
                    </div>
                </template>
                <template #after>
                    <div class="overview-transaction-amount">
                        <div class="text-income text-align-right">
                            <small class="overview-amount-label">{{ tt('Income') }}</small>
                            <small v-if="loading">0.00 USD</small>
                            <small v-else-if="!loading && transactionOverview.thisWeek && transactionOverview.thisWeek.valid">{{ getDisplayIncomeAmount(transactionOverview.thisWeek) }}</small>
                        </div>
                        <div class="text-expense text-align-right">
                            <small class="overview-amount-label">{{ tt('Expense') }}</small>
                            <small v-if="loading">0.00 USD</small>
                            <small v-else-if="!loading && transactionOverview.thisWeek && transactionOverview.thisWeek.valid">{{ getDisplayExpenseAmount(transactionOverview.thisWeek) }}</small>
                        </div>
                    </div>
                </template>
            </f7-list-item>

            <f7-list-item :link="`/transaction/list?${overviewStore.getTransactionListPageParams({ dateType: DateRange.ThisYear.type })}`" chevron-center>
                <template #title>
                    <div class="padding-top-half">
                        <span v-if="loading">This Year</span>
                        <span v-else-if="!loading">{{ tt('This Year') }}</span>
                    </div>
                </template>
                <template #after>
                    <div class="overview-transaction-amount">
                        <div class="text-income text-align-right">
                            <small class="overview-amount-label">{{ tt('Income') }}</small>
                            <small v-if="loading">0.00 USD</small>
                            <small v-else-if="!loading && transactionOverview.thisYear && transactionOverview.thisYear.valid">{{ getDisplayIncomeAmount(transactionOverview.thisYear) }}</small>
                        </div>
                        <div class="text-expense text-align-right">
                            <small class="overview-amount-label">{{ tt('Expense') }}</small>
                            <small v-if="loading">0.00 USD</small>
                            <small v-else-if="!loading && transactionOverview.thisYear && transactionOverview.thisYear.valid">{{ getDisplayExpenseAmount(transactionOverview.thisYear) }}</small>
                        </div>
                    </div>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-toolbar tabbar icons bottom class="main-tabbar">
            <f7-link class="link" href="/transaction/list">
                <f7-icon f7="square_list"></f7-icon>
                <span class="tabbar-label">{{ tt('Details') }}</span>
            </f7-link>
            <f7-link class="link" href="/account/list">
                <f7-icon f7="creditcard"></f7-icon>
                <span class="tabbar-label">{{ tt('Accounts') }}</span>
            </f7-link>
            <f7-link id="homepage-add-button" class="link" :href="isNativeApp() ? undefined : '/transaction/add'"
                     :aria-label="tt('Add Transaction')" @click="isNativeApp() && openTransactionTemplatePopover()"
                     @taphold="openTransactionTemplatePopover">
                <f7-icon f7="plus_square" class="ebk-tarbar-big-icon"></f7-icon>
            </f7-link>
            <f7-link class="link" href="/statistic/transaction">
                <f7-icon f7="chart_pie"></f7-icon>
                <span class="tabbar-label">{{ tt('Statistics') }}</span>
            </f7-link>
            <f7-link class="link" href="/settings">
                <f7-icon f7="gear_alt"></f7-icon>
                <span class="tabbar-label">{{ tt('Settings') }}</span>
            </f7-link>
        </f7-toolbar>

        <f7-popover class="template-popover-menu" target-el="#homepage-add-button"
                    v-model:opened="showTransactionTemplatePopover">
            <f7-list dividers v-if="allTransactionTemplates">
                <f7-list-item v-if="isNativeApp()" title="对话记账" link="#" @click="openChatBookkeeping">
                    <template #media><f7-icon f7="chat_bubble_2"></f7-icon></template>
                </f7-list-item>
                <f7-list-item v-if="isNativeApp()" :title="tt('Add Transaction')" link="/transaction/add" @click="showTransactionTemplatePopover = false">
                    <template #media><f7-icon f7="pencil"></f7-icon></template>
                </f7-list-item>
                <f7-list-item key="SocialGift" :title="tt('Record Social Gift')" link="/social-gifts?new=1" @click="showTransactionTemplatePopover = false">
                    <template #media><f7-icon f7="gift"></f7-icon></template>
                </f7-list-item>
                <f7-list-item key="VoiceInput" :title="tt('Voice Input')"
                              @click="addByVoiceInput"
                              v-if="isSpeechRecognitionSupported()">
                    <template #media>
                        <f7-icon f7="mic"></f7-icon>
                    </template>
                </f7-list-item>
                <f7-list-item key="AIImageRecognition" :title="tt('AI Image Recognition')"
                              @click="showAIReceiptImageRecognitionSheet = true; showTransactionTemplatePopover = false"
                              v-if="isTransactionFromAIImageRecognitionEnabled()">
                    <template #media>
                        <f7-icon f7="wand_stars"></f7-icon>
                    </template>
                </f7-list-item>
                <f7-list-item :key="template.id" :title="template.name"
                              :link="'/transaction/add?templateId=' + template.id"
                              v-for="template in allTransactionTemplates">
                    <template #media>
                        <f7-icon f7="doc_plaintext"></f7-icon>
                    </template>
                </f7-list-item>
            </f7-list>
        </f7-popover>

        <a-i-image-recognition-sheet v-model:show="showAIReceiptImageRecognitionSheet"
                                     @recognition:change="onReceiptRecognitionChanged"/>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';
import { useHomePageBase } from '@/views/base/HomePageBase.ts';

import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionTagsStore } from '@/stores/transactionTag.ts';
import { useTransactionsStore } from '@/stores/transaction.ts';
import { useTransactionTemplatesStore } from '@/stores/transactionTemplate.ts';
import { useOverviewStore } from '@/stores/overview.ts';

import { DateRange } from '@/core/datetime.ts';
import { TemplateType } from '@/core/template.ts';
import { TransactionTemplate } from '@/models/transaction_template.ts';
import type { RecognizedReceiptImageResponse } from '@/models/large_language_model.ts';

import { isNativeApp, notifyNative } from '@/lib/native_app.ts';
import { isUserLogined, isUserUnlocked } from '@/lib/userstate.ts';
import { isTransactionFromAIImageRecognitionEnabled } from '@/lib/server_settings.ts';
import { isSpeechRecognitionSupported } from '@/lib/voice.ts';
import { recognizeTransactionByVoice } from '@/lib/transaction_voice.ts';
import { getRecognizedTransactionQueryParams } from '@/lib/transaction_text.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt, getCurrentLanguageInfo } = useI18n();
const { showToast } = useI18nUIComponents();

const {
    showAmountInHomePage,
    displayDateRange,
    transactionOverview,
    getDisplayIncomeAmount,
    getDisplayExpenseAmount
} = useHomePageBase();

const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const transactionTagsStore = useTransactionTagsStore();
const transactionsStore = useTransactionsStore();
const transactionTemplatesStore = useTransactionTemplatesStore();
const overviewStore = useOverviewStore();

const loading = ref<boolean>(true);
const recognizingByVoice = ref<boolean>(false);
const showTransactionTemplatePopover = ref<boolean>(false);
const showAIReceiptImageRecognitionSheet = ref<boolean>(false);

const allTransactionTemplates = computed<TransactionTemplate[]>(() => {
    const allTemplates = transactionTemplatesStore.allVisibleTemplates;
    return allTemplates[TemplateType.Normal.type] || [];
});

function openTransactionTemplatePopover(): void {
    if (isNativeApp() || isSpeechRecognitionSupported() || isTransactionFromAIImageRecognitionEnabled() || (allTransactionTemplates.value && allTransactionTemplates.value.length)) {
        showTransactionTemplatePopover.value = true;
    }
}

function openChatBookkeeping(): void {
    showTransactionTemplatePopover.value = false;
    notifyNative('chat');
}

function init(): void {
    if (isUserLogined() && isUserUnlocked()) {
        loading.value = true;

        const promises = [
            accountsStore.loadAllAccounts({ force: false }),
            transactionCategoriesStore.loadAllCategories({ force: false }),
            transactionTagsStore.loadAllTags({ force: false }),
            transactionTemplatesStore.loadAllTemplates({ templateType: TemplateType.Normal.type,  force: false }),
            overviewStore.loadTransactionOverview({ force: false })
        ];

        Promise.all(promises).then(() => {
            loading.value = false;
        }).catch(error => {
            loading.value = false;

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    }
}

function reload(done?: () => void): void {
    const force = !!done;

    overviewStore.loadTransactionOverview({
        force: force
    }).then(() => {
        done?.();

        if (force) {
            showToast('Data has been updated');
        }
    }).catch(error => {
        done?.();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function addByVoiceInput(): void {
    if (!isSpeechRecognitionSupported()) {
        showToast('Speech recognition is not supported by this browser');
        return;
    }

    showTransactionTemplatePopover.value = false;
    recognizingByVoice.value = true;
    showToast('Listening...');
    showLoading(() => recognizingByVoice.value);

    recognizeTransactionByVoice({
        languageTag: getCurrentLanguageInfo().alternativeLanguageTag,
        allCategories: transactionCategoriesStore.allTransactionCategories,
        allVisibleAccounts: accountsStore.allVisiblePlainAccounts,
        allTags: transactionTagsStore.allTransactionTags,
        recognizeTransactionText: (text: string) => transactionsStore.recognizeTransactionText({ text })
    }).then(result => {
        const params = getRecognizedTransactionQueryParams(result);
        params.push('noTransactionDraft=true');
        props.f7router.navigate(`/transaction/add?${params.join('&')}`);
    }).catch(error => {
        showToast(error.message || error);
    }).finally(() => {
        recognizingByVoice.value = false;
        hideLoading();
    });
}

function onReceiptRecognitionChanged(result: RecognizedReceiptImageResponse): void {
    const params = getRecognizedTransactionQueryParams(result);

    params.push(`noTransactionDraft=true`);

    props.f7router.navigate(`/transaction/add?${params.join('&')}`);
}

function onPageAfterIn(): void {
    if (!loading.value) {
        reload();
    }
}

init();
</script>

<style>
.home-summary-card {
    background-color: var(--f7-card-bg-color);
    box-shadow: none;
}

.home-summary-card .home-summary-month {
    font-size: 1.3em;
}

.home-summary-card .month-expense {
    font-size: 1.5em;
}

.home-summary-card .home-summary-misc {
    opacity: 0.6;
}

.home-summary-misc > span {
    margin-inline-end: 4px;
}

.home-summary-misc > span:last-child {
    margin-inline-end: 0;
}

.home-summary-card .home-summary-header {
    position: relative;
    padding: 0;
}

.home-summary-card .home-summary-details {
    display: block;
    width: 100%;
    box-sizing: border-box;
    padding-block: 24px;
    padding-inline: 20px 64px;
    color: var(--f7-text-color);
    white-space: normal;
}

.home-summary-card .home-amount-toggle {
    position: absolute;
    top: 16px;
    inset-inline-end: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    padding: 0;
    border: 0;
    border-radius: 12px;
    background: transparent;
    color: var(--f7-text-color);
    cursor: pointer;
}

.home-summary-details:focus-visible,
.home-amount-toggle:focus-visible {
    outline: 2px solid var(--f7-theme-color);
    outline-offset: -2px;
}

.home-summary-card.card,
.ios .home-summary-card.card,
.overview-transaction-list.list.inset > ul,
.ios .overview-transaction-list.list.inset > ul {
    box-shadow: none;
}

.overview-transaction-list .item-inner {
    min-height: 76px;
}

.overview-transaction-list .overview-amount-label {
    margin-inline-end: 8px;
    color: var(--f7-text-color);
    opacity: 0.6;
}

.overview-transaction-list .item-title > div {
    overflow: hidden;
    text-overflow: ellipsis;
}

.overview-transaction-list .item-after {
    max-width: 100%;
}

.overview-transaction-list .overview-transaction-footer {
    padding-top: 6px;
    font-size: var(--ebk-large-footer-font-size);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.overview-transaction-list .overview-transaction-footer > span {
    margin-inline-end: 4px;
}

.overview-transaction-list .overview-transaction-amount {
    max-width: 100%;
}

.overview-transaction-list .overview-transaction-amount > div {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
}

.tabbar.main-tabbar .link i + span.tabbar-label {
    margin-top: var(--ebk-icon-text-margin);
}

.tabbar.main-tabbar .link i.ebk-tarbar-big-icon {
    font-size: var(--ebk-big-icon-button-size);
    width: var(--ebk-big-icon-button-size);
    height: var(--ebk-big-icon-button-size);
    line-height: var(--ebk-big-icon-button-size);
}

.template-popover-menu .popover-inner {
    max-height: 400px;
    overflow-y: auto;
}

</style>
