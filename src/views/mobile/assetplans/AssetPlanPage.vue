<template>
    <f7-page ptr @ptr:refresh="refresh">
        <f7-navbar>
            <f7-nav-left :back-link="tt('Back')" />
            <f7-nav-title :title="tt('Asset Planning')" />
            <f7-nav-right>
                <f7-link icon-f7="arrow_clockwise" :class="{ disabled: loading || reloading }" @click="refresh()" />
            </f7-nav-right>
        </f7-navbar>

        <f7-block class="margin-top">
            <f7-segmented strong>
                <f7-button :active="selectedMetric === 'net_worth'" @click="selectMetric('net_worth')">{{ tt('Net Worth') }}</f7-button>
                <f7-button :active="selectedMetric === 'total_assets'" @click="selectMetric('total_assets')">{{ tt('Total Assets') }}</f7-button>
                <f7-button :active="selectedMetric === 'total_liabilities'" @click="selectMetric('total_liabilities')">{{ tt('Total Liabilities') }}</f7-button>
            </f7-segmented>
        </f7-block>

        <template v-if="loading">
            <f7-card class="skeleton-text">
                <f7-card-header>Asset trend</f7-card-header>
                <f7-card-content class="asset-plan-mobile-skeleton">Loading chart</f7-card-content>
            </f7-card>
            <f7-list strong inset dividers class="skeleton-text">
                <f7-list-item title="Target Amount" after="0.00 USD" />
                <f7-list-item title="Current Net Worth" after="0.00 USD" />
                <f7-list-item title="Goal completion" after="0.00%" />
            </f7-list>
        </template>

        <f7-block strong inset class="asset-plan-error" v-else-if="loadError">
            <div class="text-color-red">{{ tt('Unable to retrieve asset plan') }}</div>
            <p class="text-color-gray">{{ tt(loadError) }}</p>
            <f7-button outline @click="retry">{{ tt('Retry') }}</f7-button>
        </f7-block>

        <template v-else>
            <f7-card>
                <f7-card-header>
                    <div>
                        <div>{{ metricName }}</div>
                        <small class="text-color-gray">{{ tt('Actual trend') }}</small>
                    </div>
                    <strong>{{ formatDisplayAmount(currentMetricAmount) }}</strong>
                </f7-card-header>
                <f7-card-content class="no-padding">
                    <asset-plan-chart
                        :actual-points="actualTrendPoints"
                        :currency="currency"
                        :dark="isDarkMode"
                        :plan="plan"
                        :show-plan="selectedMetric === 'net_worth'"
                    />
                </f7-card-content>
                <f7-card-footer v-if="selectedMetric !== 'net_worth'">
                    {{ tt('Plans are only available for net worth; this view shows actual history only') }}
                </f7-card-footer>
            </f7-card>

            <template v-if="selectedMetric === 'net_worth'">
                <f7-card class="asset-plan-empty text-align-center" v-if="!plan">
                    <f7-card-content>
                        <f7-icon class="asset-plan-empty-icon" f7="scope" color="blue" />
                        <h2>{{ tt('Create your net worth plan') }}</h2>
                        <p class="text-color-gray">
                            {{ tt('Set a target amount and future date to see your planned path alongside actual net worth') }}
                        </p>
                        <f7-button fill large @click="beginCreate">{{ tt('Create Plan') }}</f7-button>
                    </f7-card-content>
                </f7-card>

                <template v-else>
                    <f7-card>
                        <f7-card-header class="display-block">
                            <div class="display-flex flex-wrap gap-half">
                                <f7-chip :text="planStatusName" :color="plan.status === 'achieved' ? 'green' : (plan.status === 'not_achieved' ? 'red' : 'blue')" />
                                <f7-chip :text="comparisonStatusName" :color="plan.comparisonStatus === 'ahead' ? 'green' : (plan.comparisonStatus === 'behind' ? 'orange' : 'blue')" />
                            </div>
                            <div class="asset-plan-completion margin-top-half">{{ completionDisplay }}</div>
                            <small class="text-color-gray">{{ tt('Goal completion') }}</small>
                        </f7-card-header>
                        <f7-progressbar :progress="Number.parseFloat(plan.completionPercentage)" />
                    </f7-card>

                    <f7-list strong inset dividers>
                        <f7-list-item
                            v-for="item in planSummaryItems"
                            :key="item.title"
                            :title="item.title"
                            :after="item.value"
                        />
                    </f7-list>

                    <f7-list strong inset dividers>
                        <f7-list-button @click="beginEdit">{{ tt('Edit Plan') }}</f7-list-button>
                        <f7-list-button color="red" @click="confirmDelete">{{ tt('Delete Plan') }}</f7-list-button>
                    </f7-list>
                </template>
            </template>
        </template>

        <f7-popup v-model:opened="editing">
            <f7-page>
                <f7-navbar>
                    <f7-nav-left>
                        <f7-link :text="tt('Cancel')" :class="{ disabled: submitting }" @click="cancelEdit" />
                    </f7-nav-left>
                    <f7-nav-title :title="plan ? tt('Edit Net Worth Plan') : tt('Create Net Worth Plan')" />
                    <f7-nav-right>
                        <f7-link :text="tt('Save')" :class="{ disabled: submitting || !!validationMessage }" @click="submit" />
                    </f7-nav-right>
                </f7-navbar>

                <f7-block class="text-color-gray">
                    {{ tt('The starting amount is frozen when the plan is created') }}
                </f7-block>

                <f7-list strong inset dividers>
                    <f7-list-item
                        link="#"
                        no-chevron
                        class="list-item-with-header-and-title"
                        :header="tt('Target Amount')"
                        :title="formatDisplayAmount(targetAmount)"
                        @click="showAmountInput = true"
                    />
                    <f7-list-input
                        type="date"
                        :label="tt('Target Date')"
                        :min="minimumDate"
                        v-model:value="targetDate"
                    />
                </f7-list>

                <f7-block strong inset class="text-color-orange" v-if="validationMessage">
                    {{ tt(validationMessage) }}
                </f7-block>

                <number-pad-sheet
                    :currency="currency"
                    v-model:show="showAmountInput"
                    v-model="targetAmount"
                />
            </f7-page>
        </f7-popup>
    </f7-page>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

import AssetPlanChart from '@/components/common/AssetPlanChart.vue';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';
import { useAssetPlanPageBase } from '@/views/base/AssetPlanPageBase.ts';
import { useEnvironmentsStore } from '@/stores/environment.ts';

const { tt } = useI18n();
const { showToast, showConfirm } = useI18nUIComponents();
const environmentsStore = useEnvironmentsStore();
const showAmountInput = ref<boolean>(false);
const isDarkMode = computed<boolean>(() => environmentsStore.framework7DarkMode || false);

const {
    loading,
    reloading,
    submitting,
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
    { title: tt('Actual vs Plan'), value: formatDisplayAmount(plan.value.differenceAmount) }
] : []);

function retry(): void {
    load().catch(() => undefined);
}

function refresh(done?: () => void): void {
    reload().then(() => {
        done?.();
        showToast('Data has been updated');
    }).catch(error => {
        done?.();

        if (!error?.processed) {
            showToast(error.message || error);
        }
    });
}

function submit(): void {
    const existed = !!plan.value;

    save().then(() => {
        showToast(existed ? 'Asset plan has been saved' : 'Asset plan has been created');
    }).catch(error => {
        if (!error?.processed) {
            showToast(error.message || error);
        }
    });
}

function confirmDelete(): void {
    showConfirm('Are you sure you want to delete this asset plan? This action cannot be undone.', () => {
        remove().then(() => {
            showToast('Asset plan has been deleted');
        }).catch(error => {
            if (!error?.processed) {
                showToast(error.message || error);
            }
        });
    });
}

load().catch(() => undefined);
</script>

<style scoped>
.asset-plan-mobile-skeleton {
    min-height: 280px;
}

.asset-plan-empty {
    padding: 1rem;
}

.asset-plan-empty-icon {
    font-size: 4rem;
}

.asset-plan-completion {
    font-size: 2rem;
    font-weight: 600;
}

.asset-plan-error p {
    overflow-wrap: anywhere;
}
</style>
