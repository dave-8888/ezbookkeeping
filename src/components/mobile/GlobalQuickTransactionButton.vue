<template>
    <template v-if="isVisible">
        <form ref="panelEl" class="global-quick-transaction-mobile-panel" :style="panelStyle"
              v-show="expanded" @submit.prevent="submit">
            <div class="global-quick-transaction-mobile-field">
                <f7-icon f7="wand_stars"></f7-icon>
                <input
                    ref="inputEl"
                    type="text"
                    enterkeyhint="done"
                    :disabled="recognizingByText"
                    :placeholder="tt('Add Transaction')"
                    v-model="transactionText"
                    @keydown.esc="collapse"
                />
                <button class="global-quick-transaction-mobile-clear" type="button"
                        :aria-label="tt('Clear')" v-if="transactionText && !recognizingByText"
                        @click="transactionText = ''">
                    <f7-icon f7="xmark_circle_fill"></f7-icon>
                </button>
            </div>
            <button class="global-quick-transaction-mobile-submit" type="submit"
                    :disabled="recognizingByText">
                <f7-preloader color="white" :size="16" v-if="recognizingByText" />
                <span v-else>{{ tt('Add') }}</span>
            </button>
        </form>

        <button ref="buttonEl" class="global-quick-transaction-mobile-fab" type="button"
                :aria-label="tt('Add Transaction')" :style="buttonStyle"
                @pointerdown="onPointerDown" @click="toggle">
            <f7-preloader color="white" :size="22" v-if="recognizingByText" />
            <f7-icon :f7="expanded ? 'xmark' : 'wand_stars'" v-else></f7-icon>
        </button>
    </template>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, type CSSProperties } from 'vue';
import { f7, f7ready } from 'framework7-vue';

import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { hideLoading, showLoading, useI18nUIComponents } from '@/lib/ui/mobile.ts';

import { useUserStore } from '@/stores/user.ts';

import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

import { useDraggableEdgeFloatingButton } from '@/components/base/DraggableEdgeFloatingButtonBase.ts';
import { useQuickTransactionTextInput } from '@/components/base/QuickTransactionTextInputBase.ts';
import { getRecognizedTransactionQueryParams } from '@/lib/transaction_text.ts';
import { isUserUnlocked } from '@/lib/userstate.ts';

const FLOATING_BUTTON_SIZE = 56;
const FLOATING_BUTTON_MARGIN_X = 16;
const FLOATING_BUTTON_MARGIN_TOP = 64;
const FLOATING_BUTTON_MARGIN_BOTTOM = 88;
const FLOATING_BUTTON_GAP = 10;

const { tt } = useI18n();
const { showToast } = useI18nUIComponents();

const userStore = useUserStore();

const inputEl = ref<HTMLInputElement | null>(null);
const panelEl = ref<HTMLElement | null>(null);
const buttonEl = ref<HTMLElement | null>(null);
const expanded = ref<boolean>(false);
const routeVersion = ref<number>(0);

const isVisible = computed<boolean>(() => routeVersion.value >= 0 && aiEnabled.value && !!userStore.currentUserBasicInfo && isUserUnlocked());
const {
    edge,
    top,
    buttonStyle,
    consumeSuppressClick,
    onPointerDown
} = useDraggableEdgeFloatingButton({
    storageKey: 'ebk_mobile_global_quick_transaction_button_position',
    buttonSize: FLOATING_BUTTON_SIZE,
    marginX: FLOATING_BUTTON_MARGIN_X,
    marginTop: FLOATING_BUTTON_MARGIN_TOP,
    marginBottom: FLOATING_BUTTON_MARGIN_BOTTOM,
    onDragStart: collapse
});
const panelStyle = computed<CSSProperties>(() => {
    const style: CSSProperties = {};
    const buttonOffset = FLOATING_BUTTON_MARGIN_X + FLOATING_BUTTON_SIZE + FLOATING_BUTTON_GAP;

    if (edge.value === 'left') {
        style.top = `${top.value}px`;
        style.left = `${buttonOffset}px`;
        style.right = `${FLOATING_BUTTON_MARGIN_X}px`;
    } else if (edge.value === 'right') {
        style.top = `${top.value}px`;
        style.left = `${FLOATING_BUTTON_MARGIN_X}px`;
        style.right = `${buttonOffset}px`;
    } else {
        style.left = `${FLOATING_BUTTON_MARGIN_X}px`;
        style.right = `${FLOATING_BUTTON_MARGIN_X}px`;

        if (edge.value === 'top') {
            style.top = `${FLOATING_BUTTON_MARGIN_TOP + FLOATING_BUTTON_SIZE + FLOATING_BUTTON_GAP}px`;
        } else {
            style.bottom = `${FLOATING_BUTTON_MARGIN_BOTTOM + FLOATING_BUTTON_SIZE + FLOATING_BUTTON_GAP}px`;
        }

    }

    return style;
});

const {
    aiEnabled,
    transactionText,
    recognizingByText,
    clearTransactionText,
    submitTransactionText
} = useQuickTransactionTextInput({
    showMessage: (message: string) => showToast(message),
    showError: (error: unknown) => showToast(getErrorMessage(error)),
    onNeedManualInput: openRecognizedTransactionEditPage,
    onSaved: () => {
        expanded.value = false;
    }
});

function getErrorMessage(error: unknown): string {
    if (error && typeof error === 'object' && 'message' in error) {
        return String((error as { message?: unknown }).message || 'An error occurred');
    }

    return (error || 'An error occurred') as string;
}

function focusInput(): void {
    nextTick(() => {
        inputEl.value?.focus();
    });
}

function expand(): void {
    expanded.value = true;
    focusInput();
}

function collapse(): void {
    if (recognizingByText.value) {
        return;
    }

    expanded.value = false;
}

function toggle(): void {
    if (consumeSuppressClick()) {
        return;
    }

    if (recognizingByText.value) {
        return;
    }

    if (expanded.value) {
        collapse();
    } else {
        expand();
    }
}

function submit(): void {
    showLoading(() => recognizingByText.value);
    submitTransactionText().finally(() => {
        hideLoading();
    });
}

function getMainRouter(): Router.Router | undefined {
    const views = f7.views as unknown as {
        main?: { router: Router.Router };
        current?: { router: Router.Router };
    };

    return views.main?.router || views.current?.router;
}

function openRecognizedTransactionEditPage(result: RecognizedTransactionResponse): void {
    const params = getRecognizedTransactionQueryParams(result);
    params.push('noTransactionDraft=true');

    getMainRouter()?.navigate(`/transaction/add?${params.join('&')}`);
    clearTransactionText();
    expanded.value = false;
}

function updateVisibility(): void {
    routeVersion.value++;

    if (!isVisible.value) {
        expanded.value = false;
    }
}

function isEventInsideElement(event: PointerEvent, element: HTMLElement | null): boolean {
    const target = event.target;

    return !!element && target instanceof Node && element.contains(target);
}

function onDocumentPointerDown(event: PointerEvent): void {
    if (!expanded.value) {
        return;
    }

    if (isEventInsideElement(event, panelEl.value) || isEventInsideElement(event, buttonEl.value)) {
        return;
    }

    collapse();
}

onMounted(() => {
    f7ready((app) => {
        app.on('pageAfterIn', updateVisibility);
        updateVisibility();
    });

    document.addEventListener('pointerdown', onDocumentPointerDown, true);
});

onUnmounted(() => {
    f7.off('pageAfterIn', updateVisibility);
    document.removeEventListener('pointerdown', onDocumentPointerDown, true);
});
</script>

<style scoped>
.global-quick-transaction-mobile-panel {
    position: fixed;
    z-index: 11000;
    display: flex;
    min-width: 0;
    gap: 8px;
    padding: 8px;
    border-radius: 8px;
    background: var(--f7-card-bg-color);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.22);
}

.global-quick-transaction-mobile-field {
    display: flex;
    align-items: center;
    min-width: 0;
    flex: 1 1 auto;
    height: 42px;
    padding: 0 10px;
    border: 1px solid var(--f7-list-item-border-color);
    border-radius: 8px;
    background: var(--f7-input-bg-color, var(--f7-page-bg-color));
}

.global-quick-transaction-mobile-field > i {
    flex: 0 0 auto;
    color: var(--f7-theme-color);
}

.global-quick-transaction-mobile-field input {
    min-width: 0;
    flex: 1 1 auto;
    height: 100%;
    margin: 0 8px;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--f7-text-color);
    font: inherit;
}

.global-quick-transaction-mobile-clear,
.global-quick-transaction-mobile-submit,
.global-quick-transaction-mobile-fab {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 0;
    outline: 0;
}

.global-quick-transaction-mobile-clear {
    flex: 0 0 auto;
    padding: 0;
    background: transparent;
    color: var(--f7-text-color);
    opacity: 0.45;
}

.global-quick-transaction-mobile-submit {
    flex: 0 0 auto;
    width: auto;
    min-width: 64px;
    height: 42px;
    padding: 0 14px;
    border-radius: 8px;
    background: var(--f7-theme-color);
    color: #fff;
    font-weight: 600;
}

.global-quick-transaction-mobile-submit:disabled {
    opacity: 0.7;
}

.global-quick-transaction-mobile-fab {
    position: fixed;
    z-index: 11001;
    flex: 0 0 auto;
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: var(--f7-theme-color);
    color: #fff;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.24);
    touch-action: none;
}

.global-quick-transaction-mobile-fab i {
    font-size: 26px;
}
</style>
