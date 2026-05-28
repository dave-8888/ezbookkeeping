<template>
    <template v-if="isVisible">
        <div ref="panelEl" class="global-quick-transaction-panel" :class="edge" :style="panelStyle" v-show="expanded">
            <v-card class="pa-3" elevation="8">
                <v-form class="d-flex align-center ga-2" @submit.prevent="submit">
                    <v-text-field
                        ref="textField"
                        class="global-quick-transaction-input"
                        density="compact"
                        variant="outlined"
                        hide-details
                        clearable
                        autofocus
                        :disabled="recognizingByText"
                        :placeholder="tt('Add Transaction')"
                        v-model="transactionText"
                        @keydown.esc="collapse"
                    >
                        <template #prepend-inner>
                            <v-icon :icon="mdiMagicStaff" size="20" />
                        </template>
                    </v-text-field>
                    <v-btn color="primary" type="submit"
                           :loading="recognizingByText"
                           :disabled="recognizingByText">
                        {{ tt('Add') }}
                    </v-btn>
                </v-form>
            </v-card>
        </div>

        <v-btn ref="buttonEl" class="global-quick-transaction-fab" color="primary" size="56"
               :icon="true" :aria-label="tt('Add Transaction')" :style="buttonStyle"
               @pointerdown="onPointerDown" @click="toggle">
            <v-progress-circular indeterminate size="24" width="3" v-if="recognizingByText" />
            <v-icon :icon="expanded ? mdiClose : mdiMagicStaff" size="28" v-else />
            <v-tooltip activator="parent">{{ tt('Add Transaction') }}</v-tooltip>
        </v-btn>
    </template>

    <snack-bar ref="snackbar" />
    <edit-dialog ref="editDialog" :type="TransactionEditPageType.Transaction" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';
import EditDialog from '@/views/desktop/transactions/list/dialogs/EditDialog.vue';

import { computed, nextTick, onMounted, onUnmounted, ref, useTemplateRef, type CSSProperties } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

import { TransactionEditPageType } from '@/views/base/transactions/TransactionEditPageBase.ts';
import { useDraggableEdgeFloatingButton } from '@/components/base/DraggableEdgeFloatingButtonBase.ts';
import { useQuickTransactionTextInput } from '@/components/base/QuickTransactionTextInputBase.ts';
import { isTransactionFromAIImageRecognitionEnabled } from '@/lib/server_settings.ts';

import {
    mdiClose,
    mdiMagicStaff
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;
type EditDialogType = InstanceType<typeof EditDialog>;
type TextFieldType = {
    focus: () => void;
};
type ElementRef = HTMLElement | { $el?: HTMLElement } | null;

const FLOATING_BUTTON_SIZE = 56;
const FLOATING_BUTTON_MARGIN_X = 24;
const FLOATING_BUTTON_MARGIN_TOP = 24;
const FLOATING_BUTTON_MARGIN_BOTTOM = 24;
const FLOATING_BUTTON_GAP = 12;
const PANEL_MAX_WIDTH = 560;
const PANEL_VIEWPORT_EXTRA_WIDTH = 116;
const PANEL_DEFAULT_HEIGHT = 72;

const { tt } = useI18n();

const desktopPageStore = useDesktopPageStore();

const snackbar = useTemplateRef<SnackBarType>('snackbar');
const editDialog = useTemplateRef<EditDialogType>('editDialog');
const textField = ref<TextFieldType | null>(null);
const panelEl = ref<HTMLElement | null>(null);
const buttonEl = ref<ElementRef>(null);

const expanded = ref<boolean>(false);
const isVisible = computed<boolean>(() => isTransactionFromAIImageRecognitionEnabled());
const {
    edge,
    left,
    top,
    buttonStyle,
    consumeSuppressClick,
    onPointerDown
} = useDraggableEdgeFloatingButton({
    storageKey: 'ebk_desktop_global_quick_transaction_button_position',
    buttonSize: FLOATING_BUTTON_SIZE,
    marginX: FLOATING_BUTTON_MARGIN_X,
    marginTop: FLOATING_BUTTON_MARGIN_TOP,
    marginBottom: FLOATING_BUTTON_MARGIN_BOTTOM,
    onDragStart: collapse
});
const panelStyle = computed<CSSProperties>(() => {
    const style: CSSProperties = {};
    const offset = FLOATING_BUTTON_MARGIN_X + FLOATING_BUTTON_SIZE + FLOATING_BUTTON_GAP;
    const panelWidth = getPanelWidth();

    if (edge.value === 'left') {
        style.left = `${offset}px`;
        style.top = `${getClampedPanelTop(top.value)}px`;
    } else if (edge.value === 'right') {
        style.right = `${offset}px`;
        style.top = `${getClampedPanelTop(top.value)}px`;
    } else {
        style.left = `${getClampedPanelLeft(left.value + FLOATING_BUTTON_SIZE / 2 - panelWidth / 2, panelWidth)}px`;

        if (edge.value === 'top') {
            style.top = `${FLOATING_BUTTON_MARGIN_TOP + FLOATING_BUTTON_SIZE + FLOATING_BUTTON_GAP}px`;
        } else {
            style.bottom = `${FLOATING_BUTTON_MARGIN_BOTTOM + FLOATING_BUTTON_SIZE + FLOATING_BUTTON_GAP}px`;
        }
    }

    return style;
});

const {
    transactionText,
    recognizingByText,
    clearTransactionText,
    submitTransactionText
} = useQuickTransactionTextInput({
    showMessage: (message: string) => snackbar.value?.showMessage(message),
    showError: (error: unknown) => snackbar.value?.showError(error as Parameters<SnackBarType['showError']>[0]),
    onNeedManualInput: openRecognizedTransactionDialog,
    onSaved: () => {
        expanded.value = false;
        desktopPageStore.requestOverviewRefresh();
    }
});

function focusInput(): void {
    nextTick(() => {
        textField.value?.focus();
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
    submitTransactionText();
}

function getViewportWidth(): number {
    return window.innerWidth || document.documentElement.clientWidth || 0;
}

function getViewportHeight(): number {
    return window.innerHeight || document.documentElement.clientHeight || 0;
}

function getPanelWidth(): number {
    return Math.min(PANEL_MAX_WIDTH, Math.max(0, getViewportWidth() - PANEL_VIEWPORT_EXTRA_WIDTH));
}

function getPanelHeight(): number {
    return panelEl.value?.getBoundingClientRect().height || PANEL_DEFAULT_HEIGHT;
}

function getClampedPanelLeft(value: number, panelWidth: number): number {
    const maxLeft = Math.max(FLOATING_BUTTON_MARGIN_X, getViewportWidth() - FLOATING_BUTTON_MARGIN_X - panelWidth);

    return Math.min(Math.max(value, FLOATING_BUTTON_MARGIN_X), maxLeft);
}

function getClampedPanelTop(value: number): number {
    const maxTop = Math.max(FLOATING_BUTTON_MARGIN_TOP, getViewportHeight() - FLOATING_BUTTON_MARGIN_BOTTOM - getPanelHeight());

    return Math.min(Math.max(value, FLOATING_BUTTON_MARGIN_TOP), maxTop);
}

function openRecognizedTransactionDialog(result: RecognizedTransactionResponse): void {
    expanded.value = false;

    const editDialogPromise = editDialog.value?.open({
        time: result.time,
        type: result.type,
        categoryId: result.categoryId,
        accountId: result.sourceAccountId,
        destinationAccountId: result.destinationAccountId,
        amount: result.sourceAmount,
        destinationAmount: result.destinationAmount,
        tagIds: result.tagIds ? result.tagIds.join(',') : undefined,
        comment: result.comment,
        noTransactionDraft: true
    });

    if (!editDialogPromise) {
        return;
    }

    editDialogPromise.then(result => {
        if (result && result.message) {
            clearTransactionText();
            snackbar.value?.showMessage(result.message);
        }

        desktopPageStore.requestOverviewRefresh();
    }).catch(error => {
        if (error) {
            snackbar.value?.showError(error);
        }
    });
}

function getElement(elementRef: ElementRef): HTMLElement | null {
    if (elementRef instanceof HTMLElement) {
        return elementRef;
    }

    return elementRef?.$el instanceof HTMLElement ? elementRef.$el : null;
}

function isEventInsideElement(event: PointerEvent, elementRef: ElementRef): boolean {
    const element = getElement(elementRef);
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
    document.addEventListener('pointerdown', onDocumentPointerDown, true);
});

onUnmounted(() => {
    document.removeEventListener('pointerdown', onDocumentPointerDown, true);
});
</script>

<style scoped>
.global-quick-transaction-panel {
    position: fixed;
    z-index: 1005;
    width: min(560px, calc(100vw - 116px));
    border-radius: 8px;
}

.global-quick-transaction-input {
    min-width: 0;
}

.global-quick-transaction-fab {
    position: fixed;
    z-index: 1006;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.24);
    touch-action: none;
}

@media (max-width: 600px) {
    .global-quick-transaction-panel {
        width: calc(100vw - 116px);
    }
}
</style>
