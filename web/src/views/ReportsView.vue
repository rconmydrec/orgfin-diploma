<script setup>
import { shallowRef } from 'vue';
import { useI18n } from 'vue-i18n';

import CashFlowReport from './CashFlowReportView.vue';
import BalanceReport from './BalanceReportView.vue';
import ExpensesReport from './ExpensesReportView.vue';
import ExpenseCategorizationReport from './ExpenseCategorizationReportView.vue';
import SpendingTrendsReport from './SpendingTrendsReportView.vue';

const { t } = useI18n();

const tabs = [
  {
    id: 'cash',
    icon: 'fa-rotate-right',
    key: 'buttons.cashFlowReport',
    comp: CashFlowReport,
  },
  {
    id: 'bal',
    icon: 'fa-scale-balanced',
    key: 'buttons.balanceReport',
    comp: BalanceReport,
  },
  {
    id: 'exp',
    icon: 'fa-chart-pie',
    key: 'buttons.expensesReport',
    comp: ExpensesReport,
  },
  // AI-powered reports temporarily disabled
  // { id: 'cat',   icon: 'fa-tags',           key: 'buttons.expenseCategorizationReport', comp: ExpenseCategorizationReport },
  // { id: 'trend', icon: 'fa-chart-line',     key: 'buttons.spendingTrendsReport', comp: SpendingTrendsReport }
];

const activeId = shallowRef(tabs[0].id);
const ActiveComp = shallowRef(tabs[0].comp);

function changeTab(id) {
  if (id === activeId.value) return;
  const tab = tabs.find((t) => t.id === id);
  if (tab) {
    activeId.value = tab.id;
    ActiveComp.value = tab.comp;
  }
}
</script>

<template>
  <main class="settings-page">
    <div class="container split">
      <aside class="sidebar">
        <h2 class="title">{{ t('menu.reports') }}</h2>

        <nav class="side-nav">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            class="btn side-btn"
            :class="{ active: tab.id === activeId }"
            @click="changeTab(tab.id)">
            <i :class="['fa-solid', tab.icon]"></i>
            {{ t(tab.key) }}
          </button>
        </nav>
      </aside>

      <section class="settings-content panel">
        <component :is="ActiveComp" />
      </section>
    </div>
  </main>
</template>

<style scoped>
.side-btn {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
