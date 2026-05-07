<script setup>
import { onBeforeMount, reactive, ref, watch } from 'vue';
import { DateTime } from 'luxon';
import { useRouter } from 'vue-router';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';
import BalancesList from '@/components/reports/BalancesList.vue';
import ReportDateFilters from '@/components/reports/ReportDateFilters.vue';

const router = useRouter();

const prevDate = ref('');
const currentDate = ref('');
const prevBalanceData = reactive([]);
const currentBalanceData = reactive([]);
const accounts = reactive([]);
const selectedAccountIds = reactive([]);
const totalPrevBalance = ref(0);
const totalCurrentBalance = ref(0);
const showHiddenAccounts = ref(false);

const updateBalanceData = async (balanceDate, balancesData, totalBalance) => {
  const data = await Services.reportsService.getReport('balance', {
    accountIds: selectedAccountIds,
    balanceDate: balanceDate.value,
  });
  balancesData.length = 0;
  balancesData.push(...data);

  totalBalance.value = balancesData.reduce(
    (acc, item) => acc + item.baseCurrencyBalance,
    0,
  );
};

async function loadAccountsAndSelect() {
  try {
    const userAccounts = await Services.accountsService.getUserAccounts({
      includeHidden: showHiddenAccounts.value,
      shouldUpdate: true,
    });
    accounts.length = 0;
    if (userAccounts) {
      accounts.push(...userAccounts);
    }
  } catch (e) {
    await processError(e, router);
  }

  selectedAccountIds.length = 0;
  selectedAccountIds.push(
    ...accounts
      .filter((account) => {
        if (account.showInReports === false) return false;
        if (!showHiddenAccounts.value && account.isHidden) return false;
        return true;
      })
      .map((account) => account.id),
  );
}

async function reloadReport() {
  if (!prevDate.value || !currentDate.value) return;
  await Promise.all([
    updateBalanceData(prevDate, prevBalanceData, totalPrevBalance),
    updateBalanceData(currentDate, currentBalanceData, totalCurrentBalance),
  ]);
}

onBeforeMount(async () => {
  await loadAccountsAndSelect();

  prevDate.value = DateTime.now().minus({ month: 1 }).toISODate();
  currentDate.value = DateTime.now().toISODate();
});

watch(prevDate, async (newVal) => {
  prevDate.value = DateTime.fromISO(newVal).toISODate();
  await updateBalanceData(prevDate, prevBalanceData, totalPrevBalance);
});

watch(currentDate, async (newVal) => {
  currentDate.value = DateTime.fromISO(newVal).toISODate();
  await updateBalanceData(currentDate, currentBalanceData, totalCurrentBalance);
});

async function toggleHiddenAccounts(event) {
  showHiddenAccounts.value = event.target.checked;
  await loadAccountsAndSelect();
  await reloadReport();
}
</script>

<template>
  <div class="section-card">
    <h2>{{ $t('message.balanceReport') }}</h2>

    <ReportDateFilters
      v-model:start-date="prevDate"
      v-model:end-date="currentDate"
      start-date-label="message.prevDate"
      end-date-label="message.currentDate">
      <label class="btn btn-secondary btn-sm mb-0 align-self-end">
        <input
          type="checkbox"
          @change="toggleHiddenAccounts" />
        {{ $t('message.showHiddenAccounts') }}
      </label>
    </ReportDateFilters>
  </div>

  <div
    class="section-card"
    style="margin-top: 16px">
    <div class="balances-grid">
      <div class="balance-panel">
        <BalancesList
          :balance-data="prevBalanceData"
          :base-currency-code="currentBalanceData[0]?.baseCurrencyCode || '---'"
          :total-balance="totalPrevBalance" />
      </div>

      <div class="balance-panel">
        <BalancesList
          :balance-data="currentBalanceData"
          :base-currency-code="currentBalanceData[0]?.baseCurrencyCode || '---'"
          :total-balance="totalCurrentBalance" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.nowrap-row {
  display: flex;
  flex-wrap: nowrap;
  overflow-x: auto;
}

.col-12.col-md-6 {
  flex: 0 0 auto;
  width: 50%;
}

.balances-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  align-items: start;
}

.balance-panel {
  min-width: 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

@media (max-width: 576px) {
  .balances-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 767px) {
  .section-card {
    padding-left: 5px;
    padding-right: 5px;
  }
}
</style>
