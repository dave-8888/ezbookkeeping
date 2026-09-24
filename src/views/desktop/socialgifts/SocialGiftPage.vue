<template>
    <social-gift-content v-if="unlocked" />
    <social-gift-privacy-gate v-else :configured="configured" :busy="busy" :error="error" @unlock="unlock" @retry="status" />
</template>
<script setup lang="ts">
import { onBeforeRouteLeave } from 'vue-router';
import SocialGiftContent from './SocialGiftContent.vue';
import SocialGiftPrivacyGate from '@/components/common/SocialGiftPrivacyGate.vue';
import { useSocialGiftPrivacy } from '@/views/base/socialgifts/useSocialGiftPrivacy.ts';
const { unlocked, configured, busy, error, lock, unlock, status } = useSocialGiftPrivacy();
onBeforeRouteLeave(() => { lock(); });
</script>
