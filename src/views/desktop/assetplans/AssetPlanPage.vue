<template>
    <v-container class="asset-plan-page pa-4 pa-md-6" fluid>
        <div class="d-flex flex-wrap align-center ga-3 mb-6">
            <div>
                <h2 class="text-h4">{{ tt('Asset Planning') }}</h2>
                <p class="text-body-1 text-medium-emphasis mt-1 mb-0">{{ tt('Compare your net worth goal with actual progress') }}</p>
            </div>
            <v-spacer />
            <v-btn
                variant="text"
                :loading="reloading"
                :disabled="loading || reloading"
                :prepend-icon="mdiRefresh"
                @click="refresh"
            >
                {{ tt('Refresh') }}
            </v-btn>
        </div>

        <v-alert
            v-if="loadError"
            class="mb-5"
            type="error"
            variant="tonal"
            :title="tt('Unable to retrieve asset plan')"
        >
            <div class="mb-3">{{ tt(loadError) }}</div>
            <v-btn color="error" variant="outlined" @click="retry">{{ tt('Retry') }}</v-btn>
        </v-alert>

        <template v-if="loading">
            <v-skeleton-loader class="mb-5" type="heading, image" />
            <v-row>
                <v-col cols="12" md="4" v-for="index in 3" :key="index">
                    <v-skeleton-loader type="article" />
                </v-col>
            </v-row>
        </template>

        <template v-else-if="!loadError">
            <v-card class="mb-5" variant="outlined">
                <v-card-text class="d-flex flex-wrap align-center ga-4">
                    <v-btn-toggle
                        color="primary"
                        variant="outlined"
                        divided
                        mandatory
                        :model-value="selectedMetric"
                    >
                        <v-btn value="net_worth" @click="selectMetric('net_worth')">{{ tt('Net Worth') }}</v-btn>
                        <v-btn value="total_assets" @click="selectMetric('total_assets')">{{ tt('Total Assets') }}</v-btn>
                        <v-btn value="total_liabilities" @click="selectMetric('total_liabilities')">{{ tt('Total Liabilities') }}</v-btn>
                    </v-btn-toggle>
                    <v-spacer />
                    <div class="text-end">
                        <div class="text-caption text-medium-emphasis">{{ metricName }}</div>
                        <div class="text-h5 font-weight-medium">{{ formatDisplayAmount(currentMetricAmount) }}</div>
                    </div>
                </v-card-text>
            </v-card>

            <v-card class="mb-5" variant="outlined">
                <v-card-item>
                    <template #title>{{ metricName }} · {{ tt('Actual trend') }}</template>
                    <template #subtitle v-if="selectedMetric !== 'net_worth'">
                        {{ tt('Plans are only available for net worth; this view shows actual history only') }}
                    </template>
                </v-card-item>
                <v-divider />
                <v-card-text class="pa-2 pa-md-4">
                    <asset-plan-chart
                        :actual-points="actualTrendPoints"
                        :currency="currency"
                        :plan="plan"
                        :show-plan="selectedMetric === 'net_worth'"
                    />
                </v-card-text>
            </v-card>

            <template v-if="selectedMetric === 'net_worth'">
                <v-card v-if="!plan" class="asset-plan-empty pa-6 pa-md-10 text-center" variant="outlined">
                    <v-icon color="primary" :icon="mdiTarget" size="64" />
                    <h3 class="text-h5 mt-4">{{ tt('Create your net worth plan') }}</h3>
                    <p class="text-body-1 text-medium-emphasis mx-auto mt-2 mb-5">
                        {{ tt('Set a target amount and future date to see your planned path alongside actual net worth') }}
                    </p>
                    <v-btn color="primary" size="large" :prepend-icon="mdiPlus" @click="beginCreate">
                        {{ tt('Create Plan') }}
                    </v-btn>
                </v-card>

                <template v-else>
                    <v-card class="mb-5" variant="outlined">
                        <v-card-text class="d-flex flex-wrap align-center ga-4">
                            <div>
                                <div class="d-flex flex-wrap align-center ga-2">
                                    <v-chip :color="plan.status === 'achieved' ? 'success' : (plan.status === 'not_achieved' ? 'error' : 'primary')" variant="tonal">
                                        {{ planStatusName }}
                                    </v-chip>
                                    <v-chip :color="plan.comparisonStatus === 'ahead' ? 'success' : (plan.comparisonStatus === 'behind' ? 'warning' : 'info')" variant="tonal">
                                        {{ comparisonStatusName }}
                                    </v-chip>
                                </div>
                                <div class="text-h4 mt-3">{{ completionDisplay }}</div>
                                <div class="text-body-2 text-medium-emphasis">{{ tt('Goal completion') }}</div>
                            </div>
                            <v-spacer />
                            <v-btn variant="outlined" :prepend-icon="mdiPencilOutline" @click="beginEdit">{{ tt('Edit Plan') }}</v-btn>
                            <v-btn color="error" variant="text" :prepend-icon="mdiDeleteOutline" @click="showDeleteConfirm = true">{{ tt('Delete Plan') }}</v-btn>
                        </v-card-text>
                        <v-progress-linear
                            color="primary"
                            height="8"
                            :model-value="Number.parseFloat(plan.completionPercentage)"
                        />
                    </v-card>

                    <v-row>
                        <v-col cols="12" sm="6" lg="4" v-for="item in planSummaryItems" :key="item.title">
                            <v-card class="h-100" variant="outlined">
                                <v-card-text>
                                    <div class="text-body-2 text-medium-emphasis">{{ item.title }}</div>
                                    <div class="text-h6 mt-2" :class="item.className">{{ item.value }}</div>
                                </v-card-text>
                            </v-card>
                        </v-col>
                    </v-row>
                </template>
            </template>
        </template>

        <v-dialog width="560" persistent v-model="editing">
            <v-card class="pa-2">
                <v-card-title>{{ plan ? tt('Edit Net Worth Plan') : tt('Create Net Worth Plan') }}</v-card-title>
                <v-card-subtitle>{{ tt('The starting amount is frozen when the plan is created') }}</v-card-subtitle>
                <v-card-text>
                    <amount-input
                        class="mb-3"
                        :disabled="submitting"
                        :currency="currency"
                        :show-currency="true"
                        :label="tt('Target Amount')"
                        :placeholder="tt('Target Amount')"
                        v-model="targetAmount"
                    />
                    <v-text-field
                        type="date"
                        persistent-placeholder
                        :disabled="submitting"
                        :label="tt('Target Date')"
                        :min="minimumDate"
                        v-model="targetDate"
                    />
                    <v-alert v-if="validationMessage" type="warning" variant="tonal" density="compact">
                        {{ tt(validationMessage) }}
                    </v-alert>
                </v-card-text>
                <v-card-actions>
                    <v-spacer />
                    <v-btn :disabled="submitting" @click="cancelEdit">{{ tt('Cancel') }}</v-btn>
                    <v-btn color="primary" :loading="submitting" :disabled="!!validationMessage" @click="submit">
                        {{ tt('Save') }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog width="460" persistent v-model="showDeleteConfirm">
            <v-card>
                <v-card-title>{{ tt('Delete Net Worth Plan') }}</v-card-title>
                <v-card-text>{{ tt('Are you sure you want to delete this asset plan? This action cannot be undone.') }}</v-card-text>
                <v-card-actions>
                    <v-spacer />
                    <v-btn :disabled="deleting" @click="showDeleteConfirm = false">{{ tt('Cancel') }}</v-btn>
                    <v-btn color="error" :loading="deleting" @click="confirmDelete">{{ tt('Delete') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <snack-bar ref="snackbar" />
    </v-container>
</template>

<script setup lang="ts">
import { computed, ref, useTemplateRef } from 'vue';

import AssetPlanChart from '@/components/common/AssetPlanChart.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { useI18n } from '@/locales/helpers.ts';
import { useAssetPlanPageBase } from '@/views/base/AssetPlanPageBase.ts';

import {
    mdiDeleteOutline,
    mdiPencilOutline,
    mdiPlus,
    mdiRefresh,
    mdiTarget
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();
const snackbar = useTemplateRef<SnackBarType>('snackbar');
const showDeleteConfirm = ref<boolean>(false);

const {
    loading,
    reloading,
    submitting,
    deleting,
    editing,
    loadError,
    selectedMetric,
    targetAmount,
    targetDate,
    plan,
    currency,
    actualTrendPoints,
    currentMetricAmount,
    metricName,
    validationMessage,
    planStatusName,
    comparisonStatusName,
    completionDisplay,
    formatDisplayAmount,
    formatDisplayDate,
    load,
    reload,
    selectMetric,
    beginCreate,
    beginEdit,
    cancelEdit,
    save,
    remove
} = useAssetPlanPageBase();

const minimumDate = computed<string>(() => {
    const date = new Date();
    date.setDate(date.getDate() + 1);
    return `${date.getFullYear()}-${(date.getMonth() + 1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')}`;
});

const planSummaryItems = computed(() => plan.value ? [
    { title: tt('Target Amount'), value: formatDisplayAmount(plan.value.targetAmount) },
    { title: tt('Target Date'), value: formatDisplayDate(plan.value.targetDate) },
    { title: tt('Starting Amount'), value: formatDisplayAmount(plan.value.startAmount) },
    { title: tt('Current Net Worth'), value: formatDisplayAmount(plan.value.currentActualNetWorth) },
    { title: tt("Today's Planned Net Worth"), value: formatDisplayAmount(plan.value.todayPlannedNetWorth) },
    {
        title: tt('Actual vs Plan'),
        value: formatDisplayAmount(plan.value.differenceAmount),
        className: plan.value.differenceAmount > 0 ? 'text-success' : (plan.value.differenceAmount < 0 ? 'text-warning' : '')
    }
] : []);

function retry(): void {
    load().catch(() => undefined);
}

function refresh(): void {
    reload().then(() => {
        snackbar.value?.showMessage(tt('Data has been updated'));
    }).catch(error => {
        if (!error?.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function submit(): void {
    const existed = !!plan.value;

    save().then(() => {
        snackbar.value?.showMessage(tt(existed ? 'Asset plan has been saved' : 'Asset plan has been created'));
    }).catch(error => {
        if (!error?.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function confirmDelete(): void {
    remove().then(() => {
        showDeleteConfirm.value = false;
        snackbar.value?.showMessage(tt('Asset plan has been deleted'));
    }).catch(error => {
        if (!error?.processed) {
            snackbar.value?.showError(error);
        }
    });
}

load().catch(() => undefined);
</script>

<style scoped>
.asset-plan-page {
    max-width: 1440px;
}

.asset-plan-empty p {
    max-width: 620px;
}
</style>
