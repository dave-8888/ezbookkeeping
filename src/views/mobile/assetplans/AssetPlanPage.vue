<template>
    <f7-page>
        <f7-navbar>
            <f7-nav-left :back-link="tt('Back')" />
            <f7-nav-title :title="tt('Asset Planning')" />
        </f7-navbar>
        <monthly-asset-plan :dark="isDarkMode" />
    </f7-page>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import MonthlyAssetPlan from '@/components/common/MonthlyAssetPlan.vue';
import { useI18n } from '@/locales/helpers.ts';
import { useEnvironmentsStore } from '@/stores/environment.ts';
const { tt } = useI18n();
const environments = useEnvironmentsStore();
const isDarkMode = computed(() => environments.framework7DarkMode || false);
</script>
