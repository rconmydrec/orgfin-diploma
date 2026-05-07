<script setup>
import { onBeforeMount, reactive, ref, watch } from 'vue';
import { RouterLink, useRouter } from 'vue-router';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';
import { useUserStore } from '@/stores/user';
import Filter from '../filter/TransactionsFilter.vue';
import ListOfTransactions from './ListOfTransactions.vue';

const props = defineProps([
  'accountId',
  'isAccountDetails',
  'initialCategories',
  'startDate',
  'endDate',
]);

const userStore = useUserStore();

let transactions = reactive([]);

const loading = ref(false);
const noMoreTransactions = ref(false);
const page = ref(1);
const perPage = ref(20);

const router = useRouter();

const filterParams = reactive({
  transactionTypes: {
    expense: false,
    income: false,
    transfer: false,
  },
  accounts: [],
  fromDate: props.startDate || '',
  toDate: props.endDate || '',
  categories: props.initialCategories || [],
});

const showFilter = ref(false);
const reset = ref(true);

const returnUrlName = ref('');

if (props.isAccountDetails) {
  returnUrlName.value = 'accountDetails';
} else {
  returnUrlName.value = 'transactions';
}

async function fetchTransactions() {
  try {
    transactions.splice(0);
    await loadMoreTransactions();
  } catch (e) {
    await processError(e, router);
  }
}

watch(
  () => props.accountId,
  (newAccountId) => {
    if (newAccountId) {
      filterParams.accounts = [newAccountId];
      fetchTransactions();
    }
  },
);

onBeforeMount(async () => {
  await fetchTransactions();
});

function resetFilters() {
  page.value = 1;
  filterParams.transactionTypes = {
    expense: false,
    income: false,
    transfer: false,
  };
  filterParams.accounts = [];
  filterParams.fromDate = '';
  filterParams.toDate = '';
  filterParams.categories = [];

  noMoreTransactions.value = false;
}

async function reloadTransactions(event) {
  event.preventDefault();
  resetFilters();

  try {
    transactions.splice(0);
    await loadMoreTransactions();
    reset.value = true;
  } catch (e) {
    await processError(e, router);
  }
}

function toggleFilter(event) {
  event.preventDefault();
  showFilter.value = !showFilter.value;
}

async function loadMoreTransactions() {
  if (loading.value || noMoreTransactions.value) return;
  loading.value = true;

  try {
    if (props.accountId) {
      filterParams.accounts = [props.accountId];
    }
    const newTransactions =
      await Services.transactionsService.getUserTransactions(
        page.value,
        perPage.value,
        {
          accountId: filterParams.accounts,
          transactionTypes: filterParams.transactionTypes,
          fromDate: filterParams.fromDate,
          toDate: filterParams.toDate,
          categories: filterParams.categories,
        },
      );

    if (newTransactions.length < perPage.value) {
      noMoreTransactions.value = true;
    }

    transactions.push(...newTransactions);
    page.value++;
  } catch (e) {
    await processError(e, router);
  } finally {
    loading.value = false;
  }
}

async function filterApplied(payload) {
  if (payload.resetStatus === true) {
    resetFilters();
  } else {
    page.value = 1;
    noMoreTransactions.value = false;
    filterParams.transactionTypes = {
      ...payload.filterParams.transactionTypes,
    };
    filterParams.accounts = [...payload.filterParams.accounts].join(',');
    filterParams.fromDate = payload.filterParams.startDate;
    filterParams.toDate = payload.filterParams.toDate;
    filterParams.categories = [...payload.filterParams.categories];
  }

  transactions.splice(0);

  await loadMoreTransactions();

  reset.value = payload.resetStatus;
  showFilter.value = false;
}
</script>

<template>
  <div class="container">
    <div class="row">
      <div class="col sub-menu">
        <span v-if="userStore.isLoggedIn">
          <RouterLink
            class="btn btn-secondary"
            :to="{
              name: 'transactionNew',
              query: {
                returnUrl: returnUrlName,
                accountId: props.accountId,
              },
            }"
            :title="$t('message.new')">
            <span class="icon-text">✚</span>
          </RouterLink>
        </span>
        <span>
          <a
            href=""
            class="btn"
            :class="{
              'active-filter btn-success': !reset,
              'btn-secondary': reset,
            }"
            @click="toggleFilter"
            :title="$t('message.filter')">
            <span class="icon-text">🔍</span>
          </a>
        </span>
        <span>
          <a
            href=""
            class="btn btn-secondary"
            @click="reloadTransactions"
            :title="$t('message.refresh')">
            <span class="icon-text">↻</span>
          </a>
        </span>
      </div>
    </div>
    <div class="row">
      <div
        class="col"
        v-show="showFilter">
        <Filter
          @filter-applied="filterApplied"
          :accountId="props.accountId"
          :transactions="transactions"
          :initial-categories="filterParams.categories"
          :resetstatus="reset"
          :is-account-details="props.isAccountDetails" />
      </div>
    </div>

    <div class="row">
      <div class="col">
        <h3>{{ $t('message.yourTransactions') }}</h3>
      </div>
    </div>
    <div class="row">
      <ListOfTransactions
        :transactions="transactions"
        @load-more="loadMoreTransactions"
        :account-id="props.accountId"
        :return-url="returnUrlName" />
      <div
        v-if="loading"
        class="loading">
        Loading...
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
@use '@/assets/main.scss' as *;

.loading {
  text-align: center;
  padding: 20px;
}

.icon-text {
  font-size: 0.95rem;
  line-height: 1;
  display: inline-block;
  vertical-align: middle;
}

.sub-menu .btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.875rem;
}
</style>
