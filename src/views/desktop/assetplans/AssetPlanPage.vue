<template>
    <monthly-asset-plan :dark="isDarkMode" />
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useTheme } from 'vuetify';
import { ThemeType } from '@/core/theme.ts';
import MonthlyAssetPlan from '@/components/common/MonthlyAssetPlan.vue';
const theme = useTheme();
const isDarkMode = computed(() => theme.global.name.value === ThemeType.Dark);
</script>
