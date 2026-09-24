<template>
    <section class="privacy-gate">
        <form class="privacy-form" @submit.prevent="submit">
            <h2>{{ tt('Social gift privacy') }}</h2>
            <p>{{ tt(configured === false ? 'Set a separate passphrase to protect your contacts and gift records' : 'Enter your privacy passphrase to view contacts and gift records') }}</p>
            <template v-if="configured !== null">
                <label for="gift-passphrase">{{ tt('Privacy passphrase') }}</label>
                <input id="gift-passphrase" v-model="password" type="password" :autocomplete="configured ? 'current-password' : 'new-password'" minlength="6" maxlength="72" required :disabled="busy" />
                <template v-if="!configured">
                    <label for="gift-passphrase-confirm">{{ tt('Confirm privacy passphrase') }}</label>
                    <input id="gift-passphrase-confirm" v-model="confirmation" type="password" autocomplete="new-password" required :disabled="busy" />
                    <small>{{ tt('Use at least 6 characters and at most 72 bytes') }}</small>
                </template>
            </template>
            <p v-if="validation || error" class="privacy-error" role="alert">{{ tt(validation || error) }}</p>
            <button v-if="configured !== null" type="submit" :disabled="busy">{{ tt(busy ? 'Please wait...' : configured ? 'Unlock social gifts' : 'Set passphrase and unlock') }}</button>
            <button v-else type="button" @click="$emit('retry')">{{ tt(error ? 'Retry' : 'Loading...') }}</button>
            <small>{{ tt('Leaving this page locks it again') }}</small>
        </form>
    </section>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from '@/locales/helpers.ts';
const { tt } = useI18n();
const props = defineProps<{ configured: boolean | null; busy: boolean; error: string }>();
const emit = defineEmits<{ unlock: [password: string]; retry: [] }>();
const password = ref('');
const confirmation = ref('');
const validation = ref('');
function submit(): void {
    validation.value = '';
    const length = new TextEncoder().encode(password.value).length;
    if (Array.from(password.value).length < 6 || length > 72) { validation.value = 'Use at least 6 characters and at most 72 bytes'; return; }
    if (!props.configured && password.value !== confirmation.value) { validation.value = 'Privacy passphrases do not match'; return; }
    emit('unlock', password.value);
    password.value = '';
    confirmation.value = '';
}
</script>

<style scoped>
.privacy-gate { display: grid; place-items: center; min-height: 65vh; padding: 24px; }
.privacy-form { width: 100%; max-width: 360px; display: flex; flex-direction: column; gap: 14px; }
h2 { font-size: 24px; font-weight: 600; }
p, small { opacity: .72; line-height: 1.6; margin: 0; }
label { font-size: 14px; margin-bottom: -8px; }
input { border: 1px solid rgba(128,128,128,.4); border-radius: 10px; padding: 12px; color: inherit; background: rgba(128,128,128,.08); font: inherit; width: 100%; box-sizing: border-box; }
input:focus-visible { outline: 2px solid #2997ff; outline-offset: 2px; }
button { background: #007aff; color: white; border: 0; border-radius: 10px; padding: 12px; font: inherit; cursor: pointer; }
button:disabled { opacity: .5; cursor: wait; }
.privacy-error { color: #e5484d; opacity: 1; }
</style>
