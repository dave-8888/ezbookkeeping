import { ref } from 'vue';
import { defineStore } from 'pinia';

export const useDesktopPageStore = defineStore('desktopPages', () => {
    const showAddTransactionDialogInTransactionList = ref<boolean>(false);
    const overviewRefreshRequested = ref<boolean>(false);

    function setShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = true;
    }

    function resetShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = false;
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
        // functions
        setShowAddTransactionDialogInTransactionList,
        resetShowAddTransactionDialogInTransactionList,
        requestOverviewRefresh,
        resetOverviewRefreshRequest
    }
});
