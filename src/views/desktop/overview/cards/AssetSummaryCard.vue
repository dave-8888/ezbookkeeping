<template>
    <v-card :class="{ 'disabled': loading }">
        <template #title>
            <span>{{ tt('Asset Summary') }}</span>
        </template>

        <v-card-text>
            <div class="mb-8">
                <span class="text-body-1" v-if="!loading || (allAccounts && allAccounts.length)">{{ tt('format.misc.youHaveAccounts', { count: displayAccountCount }) }}</span>
                <v-skeleton-loader class="skeleton-no-margin mt-1 mb-2 pb-1" width="200px" type="text" :loading="true" v-else-if="loading && (!allAccounts || !allAccounts.length)"></v-skeleton-loader>
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
                            <span class="text-h5" v-if="!loading || (allAccounts && allAccounts.length)">{{ totalAssets }}</span>
                            <v-skeleton-loader class="skeleton-no-margin mt-3 mb-2" width="120px" type="text" :loading="true" v-else-if="loading && (!allAccounts || !allAccounts.length)"></v-skeleton-loader>
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
                            <span class="text-h5" style="color: rgb(var(--v-theme-error))" v-if="!loading || (allAccounts && allAccounts.length)">{{ totalLiabilities }}</span>
                            <v-skeleton-loader class="skeleton-no-margin mt-3 mb-2" width="120px" type="text" :loading="true" v-else-if="loading && (!allAccounts || !allAccounts.length)"></v-skeleton-loader>
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
                            <span class="text-h5" v-if="!loading || (allAccounts && allAccounts.length)">{{ netAssets }}</span>
                            <v-skeleton-loader class="skeleton-no-margin mt-3 mb-2" width="120px" type="text" :loading="true" v-else-if="loading && (!allAccounts || !allAccounts.length)"></v-skeleton-loader>
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

import {
    mdiBankOutline,
    mdiCreditCardOutline,
    mdiPiggyBankOutline
} from '@mdi/js';

defineProps<{
    loading: boolean;
}>();

const { tt, getCurrentNumeralSystemType } = useI18n();
const {
    allAccounts,
    netAssets,
    totalAssets,
    totalLiabilities
} = useHomePageBase();

const numeralSystem = computed(() => getCurrentNumeralSystemType());
const displayAccountCount = computed<string>(() => allAccounts.value ? numeralSystem.value.formatNumber(allAccounts.value.length) : numeralSystem.value.digitZero);
</script>
