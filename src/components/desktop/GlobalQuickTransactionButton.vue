<template>
    <template v-if="isVisible">
        <v-dialog v-model="expanded" max-width="640" :persistent="recognizingByText"
                  @after-enter="focusInput">
            <v-card class="pa-4">
                <v-card-title class="d-flex align-center justify-space-between pa-0 mb-4">
                    <span>{{ tt('Add Transaction') }}</span>
                    <v-btn variant="text" size="small" :icon="mdiClose" :aria-label="tt('Close')"
                           :disabled="recognizingByText" @click="collapse" />
                </v-card-title>
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
        </v-dialog>

        <v-btn class="global-quick-transaction-fab" color="primary" size="56"
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

import { computed, nextTick, ref, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

import { TransactionEditPageType } from '@/views/base/transactions/TransactionEditPageBase.ts';
import { useDraggableEdgeFloatingButton } from '@/components/base/DraggableEdgeFloatingButtonBase.ts';
import { useQuickTransactionTextInput } from '@/components/base/QuickTransactionTextInputBase.ts';

import {
    mdiClose,
    mdiMagicStaff
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;
type EditDialogType = InstanceType<typeof EditDialog>;
type TextFieldType = {
    focus: () => void;
};

const FLOATING_BUTTON_SIZE = 56;
const FLOATING_BUTTON_MARGIN_X = 24;
const FLOATING_BUTTON_MARGIN_TOP = 24;
const FLOATING_BUTTON_MARGIN_BOTTOM = 24;
const { tt } = useI18n();

const desktopPageStore = useDesktopPageStore();

const snackbar = useTemplateRef<SnackBarType>('snackbar');
const editDialog = useTemplateRef<EditDialogType>('editDialog');
const textField = ref<TextFieldType | null>(null);

const expanded = ref<boolean>(false);
const isVisible = computed<boolean>(() => aiEnabled.value);
const {
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
const {
    aiEnabled,
    transactionText,
    recognizingByText,
    clearTransactionText,
    submitTransactionText
} = useQuickTransactionTextInput({
    showMessage: (message: string) => snackbar.value?.showMessage(message),
    showError: (error: unknown) => snackbar.value?.showError(error as Parameters<SnackBarType['showError']>[0]),
    onNeedManualInput: openRecognizedTransactionDialog,
    onSaved: () => {
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

async function submit(): Promise<void> {
    await submitTransactionText();

    if (expanded.value) {
        focusInput();
    }
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
    }).finally(() => {
        expand();
    });
}

</script>

<style scoped>
.global-quick-transaction-input {
    min-width: 0;
}

.global-quick-transaction-fab {
    position: fixed;
    z-index: 1006;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.24);
    touch-action: none;
}

</style>
