<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card>
                <template #title>
                    <div class="title-and-toolbar d-flex align-center">
                        <v-btn class="me-3" density="compact" color="default" variant="plain"
                               :ripple="false" :icon="true" @click="goBack">
                            <v-icon :icon="mdiArrowLeft" size="24" />
                        </v-btn>
                        <span>{{ tt('Quick Account Balance Modification') }}</span>
                        <v-spacer/>
                        <v-btn density="compact" color="default" variant="text" size="24"
                               class="ms-2" :icon="true" :loading="loading" @click="loadAccounts">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="24" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </template>

                <v-card-text>
                    <asset-summary-card
                        class="mb-6"
                        :loading="loading"
                        :include-hidden-accounts="showHidden"
                        :display-account-count="summaryAccountCount"
                    />

                    <!-- 搜索和筛选区域 -->
                    <v-row class="mb-6">
                        <v-col cols="12" md="4">
                            <v-text-field
                                v-model="searchKeyword"
                                :label="tt('Search Accounts')"
                                :prepend-inner-icon="mdiMagnify"
                                clearable
                                hide-details
                                @update:model-value="filterAccounts"
                            />
                        </v-col>
                        <v-col cols="12" md="4">
                            <v-select
                                v-model="selectedCategory"
                                :label="tt('Account Category')"
                                :items="accountCategoryOptions"
                                item-title="text"
                                item-value="value"
                                clearable
                                hide-details
                                @update:model-value="filterAccounts"
                            />
                        </v-col>
                        <v-col cols="12" md="4">
                            <v-switch
                                v-model="showHidden"
                                :label="tt('Show Hidden Accounts')"
                                hide-details
                                @update:model-value="filterAccounts"
                            />
                        </v-col>
                    </v-row>

                    <!-- 批量操作工具栏 -->
                    <v-card class="mb-6" variant="outlined">
                        <v-card-text>
                            <div class="d-flex align-center flex-wrap ga-2">
                                <span class="text-subtitle-2">{{ tt('Selected') }}: {{ selectedAccounts.length }} {{ tt('accounts') }}</span>
                                <v-divider vertical class="mx-2" v-if="selectedAccounts.length > 0" />
                                <v-btn
                                    color="success"
                                    variant="tonal"
                                    :disabled="selectedAccounts.length === 0"
                                    @click="openBatchModifyDialog('add')"
                                >
                                    <v-icon :icon="mdiPlus" start />
                                    {{ tt('Batch Add Amount') }}
                                </v-btn>
                                <v-btn
                                    color="warning"
                                    variant="tonal"
                                    :disabled="selectedAccounts.length === 0"
                                    @click="openBatchModifyDialog('subtract')"
                                >
                                    <v-icon :icon="mdiMinus" start />
                                    {{ tt('Batch Subtract Amount') }}
                                </v-btn>
                                <v-btn
                                    color="info"
                                    variant="tonal"
                                    :disabled="selectedAccounts.length === 0"
                                    @click="clearSelection"
                                >
                                    <v-icon :icon="mdiClose" start />
                                    {{ tt('Clear Selection') }}
                                </v-btn>
                            </div>
                        </v-card-text>
                    </v-card>

                    <!-- 账户列表 -->
                    <v-data-table
                        v-model="selectedAccounts"
                        v-model:sort-by="sortBy"
                        :custom-key-sort="accountSorters"
                        :headers="headers"
                        :items="filteredAccounts"
                        :loading="loading"
                        show-select
                        item-value="id"
                        class="elevation-1"
                        :items-per-page="20"
                        :items-per-page-options="[10, 20, 50, 100]"
                    >
                        <template #item.name="{ item }">
                            <div class="d-flex align-center">
                                <ItemIcon
                                    size="1.5rem"
                                    icon-type="account"
                                    :icon-id="item.icon"
                                    :color="item.color"
                                    :hidden-status="item.hidden"
                                />
                                <span class="ms-2">{{ item.name }}</span>
                                <v-chip
                                    size="small"
                                    class="ms-2"
                                    v-if="item.hidden"
                                    color="grey"
                                >
                                    {{ tt('Hidden') }}
                                </v-chip>
                            </div>
                        </template>

                        <template #item.category="{ item }">
                            <div class="d-flex align-center">
                                <ItemIcon
                                    size="1rem"
                                    icon-type="account"
                                    :icon-id="getAccountCategory(item.category)?.defaultAccountIconId"
                                />
                                <span class="ms-1">{{ tt(getAccountCategory(item.category)?.name || '') }}</span>
                            </div>
                        </template>

                        <template #item.balance="{ item }">
                            <div class="d-flex flex-column align-start">
                                <span :class="item.isAsset ? 'text-income' : 'text-expense'">
                                    {{ formatAccountBalance(item.balance, item.currency) }}
                                </span>
                                <span class="text-caption text-medium-emphasis" v-if="item.currency !== cnyCurrencyCode">
                                    {{ getCurrencyName(cnyCurrencyCode) }}: {{ formatCnyEquivalent(item.balance, item.currency, !showQuickBalanceAmount) }}
                                </span>
                            </div>
                        </template>

                        <template #item.actions="{ item }">
                            <div class="d-flex ga-1">
                                <v-btn
                                    density="comfortable"
                                    color="primary"
                                    variant="text"
                                    :icon="true"
                                    @click="openModifyDialog(item)"
                                >
                                    <v-icon :icon="mdiPencil" size="18" />
                                    <v-tooltip activator="parent">{{ tt('Modify Balance') }}</v-tooltip>
                                </v-btn>
                                <v-btn
                                    density="comfortable"
                                    color="info"
                                    variant="text"
                                    :icon="true"
                                    :to="`/transaction/list?accountIds=${item.id.toString()}`"
                                >
                                    <v-icon :icon="mdiListBoxOutline" size="18" />
                                    <v-tooltip activator="parent">{{ tt('Transaction List') }}</v-tooltip>
                                </v-btn>
                            </div>
                        </template>

                        <template #no-data>
                            <div class="text-center py-6">
                                <v-icon :icon="mdiAccountSearch" size="48" class="mb-2" />
                                <p>{{ tt('No accounts found') }}</p>
                            </div>
                        </template>
                    </v-data-table>
                </v-card-text>
            </v-card>
        </v-col>
    </v-row>

    <!-- 单个账户余额修改对话框 -->
    <v-dialog v-model="showModifyDialog" max-width="500px" :persistent="modifying"
              @keydown="onAccountNavigationKeydown">
        <v-card>
            <template #title>
                <div class="d-flex align-center ga-2">
                    <span>{{ tt('Modify Account Balance') }}</span>
                    <v-spacer />
                    <v-btn :icon="mdiChevronLeft" variant="text" density="compact"
                           :aria-label="tt('Previous')" :disabled="modifying || currentAccountIndex <= 0"
                           @click="switchAccount(-1)" />
                    <span class="text-body-2 text-no-wrap">{{ currentAccountIndex + 1 }} / {{ navigationAccounts.length }}</span>
                    <v-btn :icon="mdiChevronRight" variant="text" density="compact"
                           :aria-label="tt('Next')"
                           :disabled="modifying || currentAccountIndex < 0 || currentAccountIndex >= navigationAccounts.length - 1"
                           @click="switchAccount(1)" />
                </div>
            </template>

            <v-card-text>
                <div class="mb-4">
                    <div class="d-flex align-center mb-2">
                        <ItemIcon
                            size="1.5rem"
                            icon-type="account"
                            :icon-id="currentAccount?.icon"
                            :color="currentAccount?.color"
                        />
                        <span class="ms-2 font-weight-bold">{{ currentAccount?.name }}</span>
                    </div>
                    <div class="text-body-2">
                        {{ tt('Current Balance') }}:
                        <span :class="currentAccount?.isAsset ? 'text-income' : 'text-expense'">
                            {{ formatAccountBalance(currentAccount?.balance || 0, currentAccount?.currency || 'USD') }}
                        </span>
                    </div>
                    <div class="text-body-2 text-medium-emphasis" v-if="currentAccount && currentAccount.currency !== cnyCurrencyCode">
                        {{ getCurrencyName(cnyCurrencyCode) }}:
                        {{ formatCnyEquivalent(currentAccount?.balance || 0, currentAccount?.currency || 'USD', !showQuickBalanceAmount) }}
                    </div>
                </div>

                <v-text-field
                    v-model="newBalanceExpression"
                    :disabled="modifying"
                    :label="tt('New Balance')"
                    type="text"
                    :placeholder="'1000+200-35.5'"
                    :prefix="getCurrencySymbol(currentAccount?.currency || 'USD')"
                    :rules="[newBalanceRule]"
                    :hint="newBalanceCnyHint"
                    :persistent-hint="!!newBalanceCnyHint"
                    hide-details="auto"
                    @update:model-value="updateNewBalanceFromExpression"
                    @blur="normalizeNewBalanceExpression"
                    @keydown.enter="modifySingleAccount"
                />
            </v-card-text>

            <v-card-actions>
                <v-spacer />
                <v-btn :disabled="modifying" @click="showModifyDialog = false">
                    {{ tt('Cancel') }}
                </v-btn>
                <v-btn
                    color="primary"
                    :loading="modifying"
                    @click="modifySingleAccount"
                >
                    {{ tt('Save') }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>

    <!-- 批量修改对话框 -->
    <v-dialog v-model="showBatchModifyDialog" max-width="500px">
        <v-card>
            <template #title>
                <span>{{ batchOperationType === 'add' ? tt('Batch Add Amount') : tt('Batch Subtract Amount') }}</span>
            </template>

            <v-card-text>
                <div class="mb-4">
                    <p>{{ tt('Selected accounts') }}: {{ selectedAccounts.length }}</p>
                    <v-chip-group>
                        <v-chip
                            v-for="account in selectedAccountDetails"
                            :key="account.id.toString()"
                            size="small"
                            variant="outlined"
                        >
                            {{ account.name }}
                        </v-chip>
                    </v-chip-group>
                </div>

                <v-text-field
                    v-model.number="batchAmount"
                    :label="tt('Amount')"
                    type="number"
                    step="0.01"
                    :prefix="selectedAccounts.length > 0 ? getCurrencySymbol(selectedAccountDetails[0]?.currency || 'USD') : ''"
                    :rules="[requiredRule]"
                    hide-details="auto"
                    @keydown.enter="modifyBatchAccounts"
                />

                <v-alert
                    type="info"
                    class="mt-4"
                    v-if="selectedAccounts.length > 0"
                >
                    {{ tt('All selected accounts will') }}
                    {{ batchOperationType === 'add' ? tt('increase') : tt('decrease') }}
                    {{ tt('by') }} {{ formatCurrency(batchAmount * 100, selectedAccountDetails[0]?.currency || 'USD') }}
                </v-alert>
            </v-card-text>

            <v-card-actions>
                <v-spacer />
                <v-btn @click="showBatchModifyDialog = false">
                    {{ tt('Cancel') }}
                </v-btn>
                <v-btn
                    color="primary"
                    :loading="batchModifying"
                    @click="modifyBatchAccounts"
                >
                    {{ tt('Confirm') }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from '@/locales/helpers.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import logger from '@/lib/logger.ts';
import services from '@/lib/services.ts';
import { evaluateExpressionToAmount } from '@/lib/evaluator.ts';
import { AccountCategory } from '@/core/account.ts';
import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';
import type { HiddenAmount } from '@/core/numeral.ts';
import SnackBar from '@/components/desktop/SnackBar.vue';
import AssetSummaryCard from '@/views/desktop/overview/cards/AssetSummaryCard.vue';

import {
    mdiArrowLeft,
    mdiChevronLeft,
    mdiChevronRight,
    mdiMagnify,
    mdiRefresh,
    mdiPlus,
    mdiMinus,
    mdiClose,
    mdiPencil,
    mdiListBoxOutline,
    mdiAccountSearch
} from '@mdi/js';

interface AccountItem {
    id: bigint;
    name: string;
    category: number;
    icon: number;
    color: string;
    currency: string;
    balance: number;
    hidden: boolean;
    isAsset: boolean;
    isLiability: boolean;
}

const router = useRouter();
const { tt, getCurrencyName, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const accountsStore = useAccountsStore();
const exchangeRatesStore = useExchangeRatesStore();
const settingsStore = useSettingsStore();

const cnyCurrencyCode = 'CNY';

// 状态变量
const loading = ref<boolean>(false);
const modifying = ref<boolean>(false);
const batchModifying = ref<boolean>(false);
const showModifyDialog = ref<boolean>(false);
const showBatchModifyDialog = ref<boolean>(false);

const accounts = ref<AccountItem[]>([]);
const filteredAccounts = ref<AccountItem[]>([]);
const selectedAccounts = ref<bigint[]>([]);
const searchKeyword = ref<string>('');
const selectedCategory = ref<number | null>(null);
const showHidden = ref<boolean>(false);

const sortBy = ref<{ key: string; order?: boolean | 'asc' | 'desc' }[]>([]);
const accountSorters = {
    name: (a: string, b: string) => a.localeCompare(b),
    category: (a: number, b: number) => a - b,
    currency: (a: string, b: string) => a.localeCompare(b),
    balance: (a: number, b: number) => a - b
};
const navigationAccounts = ref<AccountItem[]>([]);
const balanceDrafts = new Map<bigint, string>();
const currentAccount = ref<AccountItem | null>(null);
const currentAccountIndex = computed(() =>
    navigationAccounts.value.findIndex(account => account.id === currentAccount.value?.id)
);
const newBalance = ref<number>(0);
const newBalanceExpression = ref<string>('');
const batchAmount = ref<number>(0);
const batchOperationType = ref<'add' | 'subtract'>('add');

// 计算属性
const headers = computed(() => [
    { title: tt('Account Name'), key: 'name', sortable: true },
    { title: tt('Category'), key: 'category', sortable: true },
    { title: tt('Currency'), key: 'currency', sortable: true },
    { title: tt('Balance'), key: 'balance', sortable: true },
    { title: tt('Actions'), key: 'actions', sortable: false }
]);

const accountCategoryOptions = computed(() => {
    return AccountCategory.values().map(category => ({
        text: tt(category.name),
        value: category.type
    }));
});

const selectedAccountDetails = computed(() => {
    return accounts.value.filter(account => selectedAccounts.value.includes(account.id));
});

const summaryAccountCount = computed<number>(() => {
    if (showHidden.value) {
        return accounts.value.length;
    }

    return accounts.value.filter(account => !account.hidden).length;
});

const newBalanceCnyHint = computed<string>(() => {
    if (!currentAccount.value || currentAccount.value.currency === cnyCurrencyCode) {
        return '';
    }

    const calculatedBalance = parseNewBalanceExpression(newBalanceExpression.value);

    if (calculatedBalance === undefined) {
        return '';
    }

    const cnyEquivalent = formatCnyEquivalent(
        Math.round(calculatedBalance * 100),
        currentAccount.value.currency
    );

    return `${getCurrencyName(cnyCurrencyCode)}: ${cnyEquivalent}`;
});

// 辅助函数
const showQuickBalanceAmount = computed<boolean>(() =>
    settingsStore.appSettings.showAmountInHomePage &&
    settingsStore.appSettings.showAccountBalance
);

function formatAccountBalance(amount: number, currency: string): string {
    const displayAmount: number | HiddenAmount = showQuickBalanceAmount.value ? amount : DISPLAY_HIDDEN_AMOUNT;
    return formatAmountToLocalizedNumeralsWithCurrency(displayAmount, currency);
}

function formatCurrency(amount: number, currency: string): string {
    return formatAmountToLocalizedNumeralsWithCurrency(amount, currency);
}

function formatCnyEquivalent(amount: number, currency: string, hidden: boolean = false): string {
    if (hidden) {
        return formatAmountToLocalizedNumeralsWithCurrency(DISPLAY_HIDDEN_AMOUNT, cnyCurrencyCode);
    }

    const exchangedAmount = exchangeRatesStore.getExchangedAmount(amount, currency, cnyCurrencyCode);

    if (exchangedAmount === null) {
        return '—';
    }

    return formatAmountToLocalizedNumeralsWithCurrency(Math.trunc(exchangedAmount), cnyCurrencyCode);
}

function getCurrencySymbol(currency: string): string {
    const currencySymbols: Record<string, string> = {
        'USD': '$',
        'EUR': '€',
        'GBP': '£',
        'JPY': '¥',
        'CNY': '¥',
        'CAD': 'CA$',
        'AUD': 'AU$'
    };
    return currencySymbols[currency] || currency;
}

const requiredRule = (value: unknown) => !!value || tt('This field is required');
const newBalanceRule = (value: string) => parseNewBalanceExpression(value) !== undefined || tt('Formula is invalid');

// 方法
function goBack(): void {
    router.back();
}

function isUpToDateError(error: unknown): boolean {
    return typeof error === 'object' && error !== null &&
        'isUpToDate' in error && error.isUpToDate === true;
}

async function refreshAccountsStore(force: boolean): Promise<void> {
    try {
        await accountsStore.loadAllAccounts({ force });
    } catch (error: unknown) {
        if (isUpToDateError(error)) {
            return;
        }

        logger.warn('failed to refresh accounts store in quick balance modify page', error);
    }
}

async function refreshExchangeRates(): Promise<void> {
    try {
        await exchangeRatesStore.getLatestExchangeRates({ silent: true, force: false });
    } catch (error: unknown) {
        logger.warn('failed to refresh exchange rates in quick balance modify page', error);
    }
}

async function loadAccounts(): Promise<void> {
    loading.value = true;

    try {
        const response = await services.getAllAccounts({ visibleOnly: false });
        accounts.value = response.data.result.map(account => ({
            id: BigInt(account.id),  // 使用BigInt处理大整数ID
            name: account.name,
            category: account.category,
            icon: parseInt(account.icon),
            color: account.color,
            currency: account.currency,
            balance: account.balance,
            hidden: account.hidden,
            isAsset: account.isAsset || false,
            isLiability: account.isLiability || false
        }));

        filterAccounts();
        await refreshAccountsStore(true);
    } catch (error: unknown) {
        SnackBar['showNetworkError'](error);
    } finally {
        loading.value = false;
    }
}

function filterAccounts(): void {
    let filtered = [...accounts.value];

    // 按关键词搜索
    if (searchKeyword.value) {
        const keyword = searchKeyword.value.toLowerCase();
        filtered = filtered.filter(account =>
            account.name.toLowerCase().includes(keyword)
        );
    }

    // 按分类筛选
    if (selectedCategory.value !== null) {
        filtered = filtered.filter(account =>
            account.category === selectedCategory.value
        );
    }

    // 按隐藏状态筛选
    if (!showHidden.value) {
        filtered = filtered.filter(account => !account.hidden);
    }

    filteredAccounts.value = filtered;
}

function getAccountCategory(categoryType: number) {
    const category = AccountCategory.valueOf(categoryType);
    return category || AccountCategory.Cash;
}

function openModifyDialog(account: AccountItem): void {
    balanceDrafts.clear();
    // Freeze the filtered table order for this editing session, including other pages.
    navigationAccounts.value = [...filteredAccounts.value].sort((a, b) => {
        for (const sort of sortBy.value) {
            if (sort.order === false) continue;
            let comparison = 0;
            switch (sort.key) {
                case 'name': comparison = accountSorters.name(a.name, b.name); break;
                case 'category': comparison = accountSorters.category(a.category, b.category); break;
                case 'currency': comparison = accountSorters.currency(a.currency, b.currency); break;
                case 'balance': comparison = accountSorters.balance(a.balance, b.balance); break;
            }
            if (comparison) return sort.order === 'desc' ? -comparison : comparison;
        }
        return 0;
    });
    selectAccount(account);
    showModifyDialog.value = true;
}

function selectAccount(account: AccountItem): void {
    currentAccount.value = account;
    newBalanceExpression.value = balanceDrafts.get(account.id) ?? (account.balance / 100).toString();
    updateNewBalanceFromExpression(newBalanceExpression.value);
}

function switchAccount(direction: -1 | 1): void {
    if (modifying.value || !currentAccount.value || currentAccountIndex.value < 0) return;
    const nextAccount = navigationAccounts.value[currentAccountIndex.value + direction];
    if (!nextAccount) return;
    balanceDrafts.set(currentAccount.value.id, newBalanceExpression.value);
    selectAccount(nextAccount);
}

function onAccountNavigationKeydown(event: KeyboardEvent): void {
    if (!showModifyDialog.value || event.isComposing || event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return;
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    event.stopPropagation();
    if (!event.repeat) switchAccount(event.key === 'ArrowLeft' ? -1 : 1);
}

function parseNewBalanceExpression(expression: string): number | undefined {
    const calculatedAmount = evaluateExpressionToAmount(expression);

    if (calculatedAmount === undefined) {
        return undefined;
    }

    return calculatedAmount / 100;
}

function updateNewBalanceFromExpression(value: string): void {
    const calculatedBalance = parseNewBalanceExpression(value);

    if (calculatedBalance !== undefined) {
        newBalance.value = calculatedBalance;
    }
}

function normalizeNewBalanceExpression(): void {
    const calculatedBalance = parseNewBalanceExpression(newBalanceExpression.value);

    if (calculatedBalance !== undefined) {
        newBalance.value = calculatedBalance;
        newBalanceExpression.value = calculatedBalance.toString();
    }
}

function openBatchModifyDialog(operation: 'add' | 'subtract'): void {
    if (selectedAccounts.value.length === 0) {
        SnackBar['showError'](tt('Please select at least one account'));
        return;
    }

    batchOperationType.value = operation;
    batchAmount.value = 0;
    showBatchModifyDialog.value = true;
}

function clearSelection(): void {
    selectedAccounts.value = [];
}

async function modifySingleAccount(): Promise<void> {
    if (modifying.value) return;
    const calculatedBalance = parseNewBalanceExpression(newBalanceExpression.value);

    if (!currentAccount.value || calculatedBalance === undefined) {
        SnackBar['showError'](tt('Formula is invalid'));
        return;
    }

    newBalance.value = calculatedBalance;
    modifying.value = true;

    const accountId = currentAccount.value.id;

    try {
        const response = await services.modifyAccountBalance({
            id: accountId,
            balance: Math.round(newBalance.value * 100) // 转换为分存储
        });

        // 更新本地数据
        const accountIndex = accounts.value.findIndex(a => a.id === accountId);
        if (accountIndex !== -1 && response.data && response.data.result) {
            const accountToUpdate = accounts.value[accountIndex];
            if (accountToUpdate) {
                accountToUpdate.balance = response.data.result.balance;
            }
        }

        await loadAccounts();
        showModifyDialog.value = false;
        SnackBar['showSuccess'](tt('Account balance updated successfully'));
    } catch (error: unknown) {
        SnackBar['showNetworkError'](error);
    } finally {
        modifying.value = false;
    }
}

async function modifyBatchAccounts(): Promise<void> {
    if (selectedAccounts.value.length === 0 || batchAmount.value <= 0) {
        return;
    }

    batchModifying.value = true;

    try {
        const response = await services.modifyBatchAccountBalances({
            accountIds: selectedAccounts.value,  // 不再转换为字符串
            operation: batchOperationType.value,
            amount: Math.round(batchAmount.value * 100)
        });

        // 更新本地数据
        if (response.data && response.data.result) {
            response.data.result.forEach(modifiedAccount => {
                const accountIndex = accounts.value.findIndex(a => a.id.toString() === modifiedAccount.id);
                if (accountIndex !== -1) {
                    const accountToUpdate = accounts.value[accountIndex];
                    if (accountToUpdate) {
                        accountToUpdate.balance = modifiedAccount.balance;
                    }
                }
            });
        }

        await loadAccounts();
        selectedAccounts.value = []; // 清空选择
        showBatchModifyDialog.value = false;
        SnackBar['showSuccess'](tt('Batch account balances updated successfully'));
    } catch (error: unknown) {
        SnackBar['showNetworkError'](error);
    } finally {
        batchModifying.value = false;
    }
}

// 组件挂载时加载数据
onMounted(() => {
    loadAccounts();
    refreshExchangeRates();
});
</script>

<style scoped>
.text-income {
    color: rgb(var(--v-theme-success)) !important;
}
.text-expense {
    color: rgb(var(--v-theme-error)) !important;
}

/* 表格行悬停效果 */
:deep(.v-data-table tbody tr:hover) {
    background-color: rgba(var(--v-theme-on-surface), 0.08) !important;
}

/* 选中行的悬停效果 */
:deep(.v-data-table tbody tr.v-data-table__selected:hover) {
    background-color: rgba(var(--v-theme-primary), 0.12) !important;
}
</style>
