import { ref } from 'vue';
import { defineStore } from 'pinia';

export const useDesktopPageStore = defineStore('desktopPages', () => {
    const showAddTransactionDialogInTransactionList = ref<boolean>(false);
    const transactionListRevision = ref(0);
    const overviewRefreshRequested = ref<boolean>(false);

    function setShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = true;
    }

    function resetShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = false;
    }

    function requestTransactionListRefresh(): void {
        transactionListRevision.value++;
    }

    function requestOverviewRefresh(): void {
        overviewRefreshRequested.value = true;
    }

    function resetOverviewRefreshRequest(): void {
        overviewRefreshRequested.value = false;
    }

    return {
        // states
        showAddTransactionDialogInTransactionList,
        overviewRefreshRequested,
        transactionListRevision,
        requestTransactionListRefresh,
        // functions
        setShowAddTransactionDialogInTransactionList,
        resetShowAddTransactionDialogInTransactionList,
        requestOverviewRefresh,
        resetOverviewRefreshRequest
    }
});
