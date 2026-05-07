<script setup>
import { reactive, ref, watch } from 'vue';

import TransactionType from './TransactionType.vue';
import AccountsListContainer from '@/components/filter/AccountsListContainer.vue';
import DateFilter from '@/components/filter/DateFilter.vue';
import CategoriesFilter from '@/components/filter/CategoriesFilter.vue';

const props = defineProps([
  'transactions',
  'initialCategories',
  'resetstatus',
  'isAccountDetails',
  'accountId',
]);
const emit = defineEmits(['filterApplied']);

let filtersApplied = {};
const checkedAccounts = reactive([]);
const transactionTypes = reactive({
  expense: false,
  income: false,
  transfer: false,
});
const showCategoriesModal = ref(false);
const categories = reactive([...props['initialCategories']]);

function resetFilters() {
  filtersApplied = {}; //remove all filters
  //disable all transaction types
  for (const prop in transactionTypes) {
    transactionTypes[prop] = false;
  }
  checkedAccounts.splice(0); //remove all selected accounts
  categories.splice(0); //remove all selected categories

  updateFilteredTransactions(true);
}

watch(
  () => props['resetstatus'],
  (newReset) => {
    if (newReset) {
      resetFilters();
    }
  },
);

watch(
  () => props['accountId'],
  (newAccountId) => {
    if (newAccountId) {
      checkedAccounts.splice(0);
      checkedAccounts.push(newAccountId);
    }
  },
);

function transactionTypeChanged({ newTransactionTypes }) {
  const { expense, income, transfer } = newTransactionTypes;
  transactionTypes.expense = expense;
  transactionTypes.income = income;
  transactionTypes.transfer = transfer;
  if (expense || income || transfer) {
    filtersApplied.transactionTypes = newTransactionTypes;
  } else {
    filtersApplied.transactionTypes = null;
  }
}

function selectedAccountsUpdated({ selectedAccounts }) {
  if (selectedAccounts.length > 0) {
    filtersApplied.accounts = selectedAccounts;
    checkedAccounts.splice(0);
    checkedAccounts.push(...selectedAccounts);
  } else {
    filtersApplied.accounts = [];
  }
}

function categoriesUpdated(payload) {
  categories.splice(0);
  categories.push(...payload);
  filtersApplied.categories = categories;
  closeCategoriesModal();
}

function applyFilter(event) {
  event.preventDefault();
  updateFilteredTransactions();
}

function updateFilteredTransactions(newResetStatus = false) {
  filtersApplied.accounts = checkedAccounts;
  filtersApplied.fromDate = filtersApplied.startDate;
  filtersApplied.toDate = filtersApplied.endDate;
  filtersApplied.categories = categories;

  emit('filterApplied', {
    filterParams: filtersApplied,
    resetStatus: newResetStatus,
  });
}

function updateFilterDates({ startDate, endDate }) {
  filtersApplied.startDate = startDate;
  filtersApplied.endDate = endDate;
}

function openCategoriesModal() {
  showCategoriesModal.value = true;
}

function closeCategoriesModal() {
  showCategoriesModal.value = false;
}
</script>

<template>
  <div class="container filter-container">
    <TransactionType
      @transaction-type-changed="transactionTypeChanged"
      :types="transactionTypes" />

    <DateFilter @update-dates="updateFilterDates" />

    <div class="row">
      <div class="col">
        <AccountsListContainer
          v-show="props.isAccountDetails === false"
          @selected-accounts-updated="selectedAccountsUpdated"
          :selected-accounts="checkedAccounts" />
      </div>
    </div>

    <div class="row">
      <div class="col-12 profile-section">
        <button
          @click.stop="openCategoriesModal"
          class="btn btn-primary w-100">
          {{ $t('buttons.categories') }}
        </button>
      </div>
      <teleport to="body">
        <CategoriesFilter
          v-if="showCategoriesModal"
          :initial-categories="categories"
          @categories-updated="categoriesUpdated"
          :close-modal="closeCategoriesModal" />
      </teleport>
    </div>

    <div class="row filter-bottom-menu-row">
      <div class="col bottom-menu-container">
        <a
          href="#"
          class="btn btn-primary me-2"
          @click="applyFilter">
          Apply
        </a>
        <a
          href="#"
          class="btn btn-secondary"
          @click="resetFilters">
          Reset
        </a>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@use '@/assets/common.scss' as *;

.filter-container {
  margin-bottom: 1vh;
  background-color: $section-background-color;
  padding: 0.5rem;
}

.filter-bottom-menu-row {
  margin-top: 1vh;
}

.bottom-menu-container {
  display: flex;
  justify-content: start;
}

.row {
  margin-bottom: 1vh;
}
</style>
