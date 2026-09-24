<template>
    <f7-page :ptr="unlocked" @ptr:refresh="refresh" @page:beforeout="lock">
        <f7-navbar :title="tt('Social Gifts')" :back-link="tt('Back')" />
        <social-gift-content v-if="unlocked" ref="content" :f7route="f7route" />
        <social-gift-privacy-gate v-else :configured="configured" :busy="busy" :error="error" @unlock="unlock" @retry="status" />
    </f7-page>
</template>
<script setup lang="ts">
import { useTemplateRef } from 'vue';
import type { Router } from 'framework7/types';
import { useI18n } from '@/locales/helpers.ts';
import SocialGiftContent from './SocialGiftContent.vue';
import SocialGiftPrivacyGate from '@/components/common/SocialGiftPrivacyGate.vue';
import { useSocialGiftPrivacy } from '@/views/base/socialgifts/useSocialGiftPrivacy.ts';
defineProps<{ f7route: Router.Route }>();
const { tt } = useI18n();
const content = useTemplateRef<InstanceType<typeof SocialGiftContent>>('content');
function refresh(done: () => void): void {
    if (!content.value) { done(); return; }
    void content.value.refresh().catch(() => undefined).finally(done);
}
const { unlocked, configured, busy, error, lock, unlock, status } = useSocialGiftPrivacy();
</script>
