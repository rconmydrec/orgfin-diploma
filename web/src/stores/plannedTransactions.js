import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const usePlannedTransactionsStore = defineStore(
  'plannedTransactions',
  () => {
    // State
    const plannedTransactions = ref([]);
    const upcomingOccurrences = ref([]);
    const loading = ref(false);
    const error = ref(null);
    const futureBalance = ref(null);
    const balanceProjection = ref(null);

    // Getters
    const activeTransactions = computed(() =>
      plannedTransactions.value.filter((pt) => pt.isActive && !pt.isDeleted),
    );

    const recurringTransactions = computed(() =>
      activeTransactions.value.filter((pt) => pt.isRecurring),
    );

    const oneTimeTransactions = computed(() =>
      activeTransactions.value.filter((pt) => !pt.isRecurring),
    );

    const executedTransactions = computed(() =>
      plannedTransactions.value.filter((pt) => pt.isExecuted),
    );

    const upcomingTransactions = computed(() => {
      const now = new Date();
      return activeTransactions.value
        .filter((pt) => !pt.isExecuted && new Date(pt.plannedDate) >= now)
        .sort((a, b) => new Date(a.plannedDate) - new Date(b.plannedDate));
    });

    const overdueTransactions = computed(() => {
      const now = new Date();
      return activeTransactions.value
        .filter((pt) => !pt.isExecuted && new Date(pt.plannedDate) < now)
        .sort((a, b) => new Date(a.plannedDate) - new Date(b.plannedDate));
    });

    /**
     * Get planned transactions by account ID
     */
    const getByAccountId = computed(
      () => (accountId) =>
        activeTransactions.value.filter((pt) => pt.accountId === accountId),
    );

    /**
     * Get planned transactions in a date range
     */
    const getInDateRange = computed(() => (startDate, endDate) => {
      const start = new Date(startDate);
      const end = new Date(endDate);
      return activeTransactions.value.filter((pt) => {
        const ptDate = new Date(pt.plannedDate);
        return ptDate >= start && ptDate <= end;
      });
    });

    // Actions
    function setPlannedTransactions(transactions) {
      plannedTransactions.value = transactions;
      error.value = null;
    }

    function addPlannedTransaction(transaction) {
      plannedTransactions.value.push(transaction);
    }

    function updatePlannedTransaction(updatedTransaction) {
      const index = plannedTransactions.value.findIndex(
        (pt) => pt.id === updatedTransaction.id,
      );
      if (index !== -1) {
        plannedTransactions.value[index] = updatedTransaction;
      }
    }

    function removePlannedTransaction(id) {
      plannedTransactions.value = plannedTransactions.value.filter(
        (pt) => pt.id !== id,
      );
    }

    function setLoading(value) {
      loading.value = value;
    }

    function setError(err) {
      error.value = err;
      loading.value = false;
    }

    function clearError() {
      error.value = null;
    }

    function setFutureBalance(data) {
      futureBalance.value = data;
    }

    function setBalanceProjection(data) {
      balanceProjection.value = data;
    }

    function setUpcomingOccurrences(occurrences) {
      upcomingOccurrences.value = occurrences;
    }

    function clearPlannedTransactions() {
      plannedTransactions.value = [];
      upcomingOccurrences.value = [];
      futureBalance.value = null;
      balanceProjection.value = null;
      error.value = null;
    }

    /**
     * Calculate total planned income
     */
    const totalPlannedIncome = computed(() =>
      activeTransactions.value
        .filter((pt) => pt.isIncome && !pt.isExecuted)
        .reduce((sum, pt) => sum + parseFloat(pt.amount), 0),
    );

    /**
     * Calculate total planned expenses
     */
    const totalPlannedExpenses = computed(() =>
      activeTransactions.value
        .filter((pt) => !pt.isIncome && !pt.isExecuted)
        .reduce((sum, pt) => sum + parseFloat(pt.amount), 0),
    );

    /**
     * Get statistics for dashboard
     */
    const statistics = computed(() => ({
      total: activeTransactions.value.length,
      upcoming: upcomingTransactions.value.length,
      overdue: overdueTransactions.value.length,
      recurring: recurringTransactions.value.length,
      executed: executedTransactions.value.length,
      totalIncome: totalPlannedIncome.value,
      totalExpenses: totalPlannedExpenses.value,
    }));

    return {
      // State
      plannedTransactions,
      upcomingOccurrences,
      loading,
      error,
      futureBalance,
      balanceProjection,

      // Getters
      activeTransactions,
      recurringTransactions,
      oneTimeTransactions,
      executedTransactions,
      upcomingTransactions,
      overdueTransactions,
      getByAccountId,
      getInDateRange,
      totalPlannedIncome,
      totalPlannedExpenses,
      statistics,

      // Actions
      setPlannedTransactions,
      addPlannedTransaction,
      updatePlannedTransaction,
      removePlannedTransaction,
      setLoading,
      setError,
      clearError,
      setFutureBalance,
      setBalanceProjection,
      setUpcomingOccurrences,
      clearPlannedTransactions,
    };
  },
);
