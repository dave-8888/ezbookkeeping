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
                        <span>{{ tt('Quick Balance Modification') }}</span>
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
                            <span :class="item.isAsset ? 'text-income' : 'text-expense'">
                                {{ formatCurrency(item.balance, item.currency) }}
                            </span>
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
    <v-dialog v-model="showModifyDialog" max-width="500px">
        <v-card>
            <template #title>
                <span>{{ tt('Modify Account Balance') }}</span>
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
                            {{ formatCurrency(currentAccount?.balance || 0, currentAccount?.currency || 'USD') }}
                        </span>
                    </div>
                </div>

                <v-text-field
                    v-model.number="newBalance"
                    :label="tt('New Balance')"
                    type="number"
                    step="0.01"
                    :prefix="getCurrencySymbol(currentAccount?.currency || 'USD')"
                    :rules="[requiredRule]"
                    hide-details="auto"
                />
            </v-card-text>

            <v-card-actions>
                <v-spacer />
                <v-btn @click="showModifyDialog = false">
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
import services from '@/lib/services.ts';
import { AccountCategory } from '@/core/account.ts';
import SnackBar from '@/components/desktop/SnackBar.vue';

import {
    mdiArrowLeft,
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
const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

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

const currentAccount = ref<AccountItem | null>(null);
const newBalance = ref<number>(0);
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

// 辅助函数
function formatCurrency(amount: number, currency: string): string {
    return formatAmountToLocalizedNumeralsWithCurrency(amount, currency);
}

function getCurrencySymbol(currency: string): string {
    // 简单的货币符号映射
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

// 验证规则
const requiredRule = (value: any) => !!value || tt('This field is required');
// 方法
function goBack(): void {
    router.back();
}

async function loadAccounts(): Promise<void> {
    loading.value = true;

    try {
        const response = await services.getAllAccounts({ visibleOnly: false });
        accounts.value = response.data.result.map((account: any) => ({
            id: BigInt(account.id),  // 使用BigInt处理大整数ID
            name: account.name,
            category: account.category,
            icon: parseInt(account.icon),
            color: account.color,
            currency: account.currency,
            balance: account.balance,
            hidden: account.hidden,
            isAsset: account.isAsset,
            isLiability: account.isLiability
        }));

        filterAccounts();
    } catch (error: any) {
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
    currentAccount.value = account;
    newBalance.value = account.balance / 100; // 转换为小数形式显示
    showModifyDialog.value = true;
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
    console.log('currentAccount',currentAccount)
    if (!currentAccount.value || newBalance.value === null || newBalance.value === undefined) {
        return;
    }

    modifying.value = true;

    // 调试信息
    console.log('Sending modify request:', {
        id: currentAccount.value.id,
        idString: currentAccount.value.id.toString(),
        idNumber: Number(currentAccount.value.id),
        balance: Math.round(newBalance.value * 100),
        idType: typeof currentAccount.value.id,
        balanceType: typeof Math.round(newBalance.value * 100),
        isSafeInteger: Number.isSafeInteger(currentAccount.value.id),
        maxSafeInteger: Number.MAX_SAFE_INTEGER
    });

    try {
        const response = await services.modifyAccountBalance({
            id: currentAccount.value.id,
            balance: Math.round(newBalance.value * 100) // 转换为分存储
        });

        // 更新本地数据
        const accountIndex = accounts.value.findIndex(a => a.id === currentAccount.value!.id);
        if (accountIndex !== -1 && response.data && response.data.result) {
            const accountToUpdate = accounts.value[accountIndex];
            if (accountToUpdate) {
                accountToUpdate.balance = response.data.result.balance;
            }
        }

        filterAccounts(); // 重新过滤以更新显示
        showModifyDialog.value = false;
        SnackBar['showSuccess'](tt('Account balance updated successfully'));
    } catch (error: any) {
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
            response.data.result.forEach((modifiedAccount: any) => {
                const accountIndex = accounts.value.findIndex(a => a.id.toString() === modifiedAccount.id);
                if (accountIndex !== -1) {
                    const accountToUpdate = accounts.value[accountIndex];
                    if (accountToUpdate) {
                        accountToUpdate.balance = modifiedAccount.balance;
                    }
                }
            });
        }

        filterAccounts(); // 重新过滤以更新显示
        selectedAccounts.value = []; // 清空选择
        showBatchModifyDialog.value = false;
        SnackBar['showSuccess'](tt('Batch account balances updated successfully'));
    } catch (error: any) {
        SnackBar['showNetworkError'](error);
    } finally {
        batchModifying.value = false;
    }
}

// 组件挂载时加载数据
onMounted(() => {
    loadAccounts();
});
</script>

<style scoped>
.text-income {
    color: rgb(var(--v-theme-success)) !important;
}
.text-expense {
    color: rgb(var(--v-theme-error)) !important;
}
</style>
