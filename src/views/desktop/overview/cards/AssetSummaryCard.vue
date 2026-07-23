<template>
    <v-card :class="{ 'disabled': loading }">
        <template #title>
            <span>{{ tt('Asset Summary') }}</span>
        </template>

        <v-card-text>
            <div class="mb-8">
                <span class="text-body-1" v-if="!loading || hasDisplayAccounts">{{ tt('format.misc.youHaveAccounts', { count: displayAccountCountText }) }}</span>
                <v-skeleton-loader class="skeleton-no-margin mt-1 mb-2 pb-1" width="200px" type="text" :loading="true" v-else-if="loading && !hasDisplayAccounts"></v-skeleton-loader>
            </div>

            <v-row>
                <v-col cols="12" md="4">
                    <div class="d-flex align-center">
                        <div class="me-3">
                            <v-avatar rounded color="grey" size="42" class="elevation-1">
                                <v-icon size="24" :icon="mdiBankOutline"/>
                            </v-avatar>
                        </div>

                        <div class="d-flex flex-column">
                            <span class="text-caption">{{ tt('Total assets') }}</span>
                            <span class="text-h5" v-if="!loading || hasDisplayAccounts">{{ totalAssets }}</span>
                            <v-skeleton-loader class="skeleton-no-margin mt-3 mb-2" width="120px" type="text" :loading="true" v-else-if="loading && !hasDisplayAccounts"></v-skeleton-loader>
                        </div>
                    </div>
                </v-col>

                <v-col cols="12" md="4">
                    <div class="d-flex align-center">
                        <div class="me-3">
                            <v-avatar rounded color="expense" size="42" class="elevation-1">
                                <v-icon size="24" :icon="mdiCreditCardOutline"/>
                            </v-avatar>
                        </div>

                        <div class="d-flex flex-column">
                            <span class="text-caption">{{ tt('Total liabilities') }}</span>
                            <span class="text-h5" style="color: rgb(var(--v-theme-error))" v-if="!loading || hasDisplayAccounts">{{ totalLiabilities }}</span>
                            <v-skeleton-loader class="skeleton-no-margin mt-3 mb-2" width="120px" type="text" :loading="true" v-else-if="loading && !hasDisplayAccounts"></v-skeleton-loader>
                        </div>
                    </div>
                </v-col>

                <v-col cols="12" md="4">
                    <div class="d-flex align-center">
                        <div class="me-3">
                            <v-avatar rounded color="primary" size="42" class="elevation-1">
                                <v-icon size="24" :icon="mdiPiggyBankOutline"/>
                            </v-avatar>
                        </div>

                        <div class="d-flex flex-column">
                            <span class="text-caption">{{ tt('Net assets') }}</span>
                            <span class="text-h5" v-if="!loading || hasDisplayAccounts">{{ netAssets }}</span>
                            <v-skeleton-loader class="skeleton-no-margin mt-3 mb-2" width="120px" type="text" :loading="true" v-else-if="loading && !hasDisplayAccounts"></v-skeleton-loader>
                        </div>
                    </div>
                </v-col>
            </v-row>
        </v-card-text>
    </v-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useHomePageBase } from '@/views/base/HomePageBase.ts';
import { useAccountsStore } from '@/stores/account.ts';

import {
    mdiBankOutline,
    mdiCreditCardOutline,
    mdiPiggyBankOutline
} from '@mdi/js';

const props = withDefaults(defineProps<{
    loading: boolean;
    includeHiddenAccounts?: boolean;
    displayAccountCount?: number;
}>(), {
    includeHiddenAccounts: false
});

const { tt, getCurrentNumeralSystemType, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const accountsStore = useAccountsStore();
const {
    allAccounts,
    defaultCurrency,
    showAmountInHomePage
} = useHomePageBase();

const numeralSystem = computed(() => getCurrentNumeralSystemType());
const displayAccountCountValue = computed<number>(() => props.displayAccountCount ?? allAccounts.value?.length ?? 0);
const hasDisplayAccounts = computed<boolean>(() => displayAccountCountValue.value > 0);
const displayAccountCountText = computed<string>(() => numeralSystem.value.formatNumber(displayAccountCountValue.value));
const netAssets = computed<string>(() => {
    return formatAmountToLocalizedNumeralsWithCurrency(
        accountsStore.getNetAssets(showAmountInHomePage.value, props.includeHiddenAccounts),
        defaultCurrency.value
    );
});
const totalAssets = computed<string>(() => {
    return formatAmountToLocalizedNumeralsWithCurrency(
        accountsStore.getTotalAssets(showAmountInHomePage.value, props.includeHiddenAccounts),
        defaultCurrency.value
    );
});
const totalLiabilities = computed<string>(() => {
    return formatAmountToLocalizedNumeralsWithCurrency(
        accountsStore.getTotalLiabilities(showAmountInHomePage.value, props.includeHiddenAccounts),
        defaultCurrency.value
    );
});
</script>
