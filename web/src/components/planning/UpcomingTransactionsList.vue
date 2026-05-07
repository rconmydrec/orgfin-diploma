<template>
  <div class="upcoming-transactions-list">
    <div
      v-if="groupedTransactions.length === 0"
      class="empty-state">
      <i class="bi bi-calendar-check display-1 text-muted mb-3"></i>
      <p class="text-muted">
        {{ $t('financialPlanning.noPlannedTransactions') }}
      </p>
    </div>

    <div
      v-for="group in groupedTransactions"
      :key="group.date"
      class="transaction-group">
      <div
        class="group-header"
        :class="group.class"
        @click="toggleGroup(group.date)">
        <i
          class="collapse-icon fa-solid"
          :class="
            collapsedGroups[group.date]
              ? 'fa-circle-chevron-right'
              : 'fa-circle-chevron-down'
          "></i>
        <i :class="group.icon"></i>
        <span class="group-title">{{ group.title }}</span>
        <span class="group-count">{{ group.transactions.length }}</span>
      </div>

      <!-- Totals row (always visible) -->
      <div
        v-if="group.totals"
        class="group-totals-summary"
        @click="toggleGroup(group.date)">
        <div class="totals-row">
          <span class="totals-label">{{ $t('financialPlanning.total') }}:</span>
          <div class="totals-amounts">
            <span
              v-if="group.totals.income > 0"
              class="total-income">
              +{{ formatCurrency(group.totals.income) }}
            </span>
            <span
              v-if="group.totals.expenses > 0"
              class="total-expenses">
              -{{ formatCurrency(group.totals.expenses) }}
            </span>
            <span
              class="total-net"
              :class="{
                positive: group.totals.net >= 0,
                negative: group.totals.net < 0,
              }">
              {{ group.totals.net >= 0 ? '+' : ''
              }}{{ formatCurrency(group.totals.net) }}
            </span>
          </div>
        </div>
      </div>

      <!-- Collapsible transactions list -->
      <div
        v-show="!collapsedGroups[group.date]"
        class="transactions-list">
        <div
          v-for="transaction in group.transactions"
          :key="`${transaction.plannedTransactionId}-${transaction.occurrenceDate}`"
          class="transaction-card"
          :class="{
            overdue: group.isOverdue,
            inactive: transaction.isActive === false,
          }">
          <div
            class="transaction-icon"
            :class="transaction.isIncome ? 'income' : 'expense'">
            <i
              :class="
                transaction.isIncome
                  ? 'bi bi-arrow-up-circle'
                  : 'bi bi-arrow-down-circle'
              "></i>
          </div>

          <div class="transaction-info">
            <div class="transaction-header">
              <span class="transaction-label">
                {{ transaction.label || $t('common.noLabel') }}
              </span>
              <span
                v-if="transaction.isRecurring"
                class="recurring-badge"
                :title="$t('financialPlanning.recurring')">
                <i class="fa-solid fa-arrows-rotate"></i>
              </span>
              <span
                v-if="transaction.isActive === false"
                class="inactive-badge">
                <i class="bi bi-pause-circle"></i>
                {{ $t('financialPlanning.inactive') }}
              </span>
            </div>
            <div class="transaction-details">
              <span class="transaction-date">
                <i class="bi bi-calendar3"></i>
                {{ formatDate(transaction.occurrenceDate) }}
              </span>
            </div>
          </div>

          <div
            class="transaction-amount"
            :class="transaction.isIncome ? 'income' : 'expense'">
            <span class="amount-value">
              {{ transaction.isIncome ? '+' : '-'
              }}{{ formatCurrency(transaction.amount) }}
            </span>
          </div>

          <div class="transaction-actions">
            <i
              @click="$emit('edit', transaction.plannedTransactionId)"
              class="fa-solid fa-pen-to-square action-icon edit-icon"
              :title="$t('common.edit')"></i>
            <i
              @click="$emit('delete', transaction.plannedTransactionId)"
              class="fa-solid fa-trash-can action-icon delete-icon"
              :title="$t('common.delete')"></i>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, reactive } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
  transactions: {
    type: Array,
    required: true,
  },
  currency: {
    type: String,
    default: 'USD',
  },
  period: {
    type: String,
    default: 'daily',
    validator: (value) => ['daily', 'weekly', 'monthly'].includes(value),
  },
  startDate: {
    type: String,
    default: '',
  },
  endDate: {
    type: String,
    default: '',
  },
});

defineEmits(['edit', 'delete']);

// State for collapsed groups - all groups collapsed by default
const collapsedGroups = reactive({});

// Toggle group collapse state
const toggleGroup = (groupDate) => {
  collapsedGroups[groupDate] = !collapsedGroups[groupDate];
};

// Helper functions
function getWeekStart(date) {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1); // Adjust when day is Sunday
  return new Date(d.setDate(diff));
}

function getWeekEnd(date) {
  const start = getWeekStart(date);
  const end = new Date(start);
  end.setDate(end.getDate() + 6);
  return end;
}

function getMonthStart(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

function getMonthEnd(date) {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}

function formatPeriodTitle(startDate, endDate, period) {
  const options = { month: 'short', day: 'numeric' };
  const start = startDate.toLocaleDateString('en-US', options);
  const end = endDate.toLocaleDateString('en-US', options);

  if (period === 'daily') {
    return startDate.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      weekday: 'short',
    });
  } else if (period === 'weekly') {
    return `${start} - ${end}`;
  } else if (period === 'monthly') {
    return startDate.toLocaleDateString('en-US', {
      month: 'long',
      year: 'numeric',
    });
  }
  return `${start} - ${end}`;
}

function calculateGroupTotal(transactions) {
  let totalIncome = 0;
  let totalExpenses = 0;

  transactions.forEach((tx) => {
    if (tx.isIncome) {
      totalIncome += parseFloat(tx.amount || 0);
    } else {
      totalExpenses += parseFloat(tx.amount || 0);
    }
  });

  return {
    income: totalIncome,
    expenses: totalExpenses,
    net: totalIncome - totalExpenses,
  };
}

const groupedTransactions = computed(() => {
  const now = new Date();
  now.setHours(0, 0, 0, 0);
  const today = new Date(now);

  // Initialize collapsed state for all groups

  // Filter transactions by date range if filters are set
  let filteredTransactions = props.transactions;

  if (props.startDate || props.endDate) {
    filteredTransactions = props.transactions.filter((tx) => {
      const txDate = new Date(tx.occurrenceDate);
      txDate.setHours(0, 0, 0, 0);

      let passesFilter = true;

      if (props.startDate) {
        const startDate = new Date(props.startDate);
        startDate.setHours(0, 0, 0, 0);
        passesFilter = passesFilter && txDate >= startDate;
      }

      if (props.endDate) {
        const endDate = new Date(props.endDate);
        endDate.setHours(0, 0, 0, 0);
        passesFilter = passesFilter && txDate <= endDate;
      }

      return passesFilter;
    });
  }

  // Calculate week boundaries
  const currentWeekEnd = getWeekEnd(today);
  const nextWeekStart = new Date(currentWeekEnd);
  nextWeekStart.setDate(nextWeekStart.getDate() + 1);
  const nextWeekEnd = getWeekEnd(nextWeekStart);

  // Initialize groups
  const groups = {
    overdue: [],
    today: [],
    thisWeek: [],
    nextWeek: [],
    later: [],
  };

  // Group transactions
  filteredTransactions.forEach((tx) => {
    const txDate = new Date(tx.occurrenceDate);
    txDate.setHours(0, 0, 0, 0);

    if (txDate < today) {
      groups.overdue.push(tx);
    } else if (txDate.getTime() === today.getTime()) {
      groups.today.push(tx);
    } else if (txDate > today && txDate <= currentWeekEnd) {
      groups.thisWeek.push(tx);
    } else if (txDate >= nextWeekStart && txDate <= nextWeekEnd) {
      groups.nextWeek.push(tx);
    } else {
      groups.later.push(tx);
    }
  });

  const result = [];

  // Overdue
  if (groups.overdue.length > 0) {
    const sorted = groups.overdue.sort(
      (a, b) => new Date(a.occurrenceDate) - new Date(b.occurrenceDate),
    );
    result.push({
      date: 'overdue',
      title: t('financialPlanning.overdue'),
      icon: 'bi bi-exclamation-triangle-fill',
      class: 'overdue',
      isOverdue: true,
      transactions: sorted,
      totals: calculateGroupTotal(sorted),
    });
  }

  // Today
  if (groups.today.length > 0) {
    result.push({
      date: 'today',
      title: t('financialPlanning.today'),
      icon: 'bi bi-calendar-day',
      class: 'today',
      isOverdue: false,
      transactions: groups.today,
      totals: calculateGroupTotal(groups.today),
    });
  }

  // This Week (remaining days)
  if (groups.thisWeek.length > 0) {
    const sorted = groups.thisWeek.sort(
      (a, b) => new Date(a.occurrenceDate) - new Date(b.occurrenceDate),
    );
    result.push({
      date: 'thisWeek',
      title: t('financialPlanning.thisWeek'),
      icon: 'bi bi-calendar-week',
      class: 'this-week',
      isOverdue: false,
      transactions: sorted,
      totals: calculateGroupTotal(sorted),
    });
  }

  // Next Week
  if (groups.nextWeek.length > 0) {
    const sorted = groups.nextWeek.sort(
      (a, b) => new Date(a.occurrenceDate) - new Date(b.occurrenceDate),
    );
    result.push({
      date: 'nextWeek',
      title: t('financialPlanning.nextWeek'),
      icon: 'bi bi-calendar-week',
      class: 'next-week',
      isOverdue: false,
      transactions: sorted,
      totals: calculateGroupTotal(sorted),
    });
  }

  // Later - split by period
  if (groups.later.length > 0) {
    const laterGroups = groupByPeriod(groups.later, nextWeekEnd, props.period);
    laterGroups.forEach((group) => {
      result.push(group);
    });
  }

  // Initialize collapsed state for new groups
  result.forEach((group) => {
    if (!(group.date in collapsedGroups)) {
      collapsedGroups[group.date] = true; // collapsed by default
    }
  });

  return result;
});

function groupByPeriod(transactions, afterDate, period) {
  const groups = new Map();

  transactions.forEach((tx) => {
    const txDate = new Date(tx.occurrenceDate);
    txDate.setHours(0, 0, 0, 0);

    let periodKey;
    let periodStart;
    let periodEnd;

    if (period === 'daily') {
      periodKey = txDate.toISOString().split('T')[0];
      periodStart = new Date(txDate);
      periodEnd = new Date(txDate);
    } else if (period === 'weekly') {
      periodStart = getWeekStart(txDate);
      periodEnd = getWeekEnd(txDate);
      periodKey = periodStart.toISOString().split('T')[0];
    } else if (period === 'monthly') {
      periodStart = getMonthStart(txDate);
      periodEnd = getMonthEnd(txDate);
      periodKey = `${txDate.getFullYear()}-${txDate.getMonth()}`;
    }

    if (!groups.has(periodKey)) {
      groups.set(periodKey, {
        key: periodKey,
        startDate: periodStart,
        endDate: periodEnd,
        transactions: [],
      });
    }

    groups.get(periodKey).transactions.push(tx);
  });

  // Convert to array and sort by date
  const result = Array.from(groups.values())
    .sort((a, b) => a.startDate - b.startDate)
    .map((group) => {
      const sorted = group.transactions.sort(
        (a, b) => new Date(a.occurrenceDate) - new Date(b.occurrenceDate),
      );
      return {
        date: group.key,
        title: formatPeriodTitle(group.startDate, group.endDate, period),
        icon: 'bi bi-calendar-range',
        class: 'later',
        isOverdue: false,
        transactions: sorted,
        totals: calculateGroupTotal(sorted),
        startDate: group.startDate,
        endDate: group.endDate,
      };
    });

  return result;
}

function formatDate(dateString) {
  return new Date(dateString).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

function formatCurrency(amount) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: props.currency,
  }).format(amount);
}
</script>

<style scoped>
.upcoming-transactions-list {
  margin-top: 2rem;
}

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
}

.transaction-group {
  margin-bottom: 2rem;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background: #f8f9fa;
  border-radius: 8px 8px 0 0;
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  transition: background-color 0.2s;
  user-select: none;
}

.group-header:hover {
  background: #e9ecef;
}

.group-header:hover .collapse-icon {
  color: #0d6efd;
  transform: scale(1.2);
}

.group-header.overdue:hover {
  background: #ffe9a8;
}

.group-header.today:hover {
  background: #b8dde4;
}

.group-header.tomorrow:hover {
  background: #d1d3d5;
}

.collapse-icon {
  font-size: 1.5rem;
  color: #0d6efd;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.group-header:hover .collapse-icon {
  color: #0a58ca;
  transform: scale(1.3);
}

.group-header.overdue {
  background: #fff3cd;
  color: #664d03;
}

.group-header.today {
  background: #d1ecf1;
  color: #0c5460;
}

.group-header.tomorrow {
  background: #e2e3e5;
  color: #383d41;
}

.group-count {
  margin-left: auto;
  background: white;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
}

.transactions-list {
  background: white;
  border: 1px solid #dee2e6;
  border-top: none;
  border-radius: 0 0 8px 8px;
}

.group-totals-summary {
  background: #f8f9fa;
  border: 1px solid #dee2e6;
  border-top: none;
  border-radius: 0 0 8px 8px;
  padding: 0.75rem 1rem;
  cursor: pointer;
  transition: background-color 0.2s;
  user-select: none;
}

.group-totals-summary:hover {
  background: #e9ecef;
}

.totals-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.totals-label {
  font-size: 0.875rem;
  color: #495057;
}

.totals-amounts {
  display: flex;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
}

.total-income {
  color: #198754;
  font-size: 0.875rem;
}

.total-expenses {
  color: #dc3545;
  font-size: 0.875rem;
}

.total-net {
  font-size: 1rem;
  font-weight: 700;
  padding: 0.25rem 0.75rem;
  border-radius: 6px;
  background: white;
}

.total-net.positive {
  color: #198754;
}

.total-net.negative {
  color: #dc3545;
}

.transaction-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  border-bottom: 1px solid #f1f3f5;
  transition: background-color 0.2s;
}

.transaction-card:last-child {
  border-bottom: none;
}

.transaction-card:hover {
  background-color: #f8f9fa;
}

.transaction-card.overdue {
  background-color: #fff9e6;
}

.transaction-card.inactive {
  opacity: 0.5;
  background-color: #f8f9fa;
}

.transaction-icon {
  font-size: 1.5rem;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.transaction-icon.income {
  background-color: #d1f2eb;
  color: #198754;
}

.transaction-icon.expense {
  background-color: #f8d7da;
  color: #dc3545;
}

.transaction-info {
  flex: 1;
  min-width: 0;
}

.transaction-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}

.transaction-label {
  font-weight: 600;
  font-size: 1rem;
}

.recurring-badge {
  font-size: 0.875rem;
  background: #e7f1ff;
  color: #0d6efd;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: help;
  transition: all 0.2s ease;
}

.recurring-badge:hover {
  background: #0d6efd;
  color: white;
  transform: rotate(180deg) scale(1.1);
}

.inactive-badge {
  font-size: 0.75rem;
  background: #e9ecef;
  color: #6c757d;
  padding: 0.125rem 0.5rem;
  border-radius: 12px;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.transaction-details {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  font-size: 0.875rem;
  color: #6c757d;
}

.transaction-details > span {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.transaction-amount {
  font-size: 1.25rem;
  font-weight: 700;
  min-width: 120px;
  text-align: right;
}

.transaction-amount.income {
  color: #198754;
}

.transaction-amount.expense {
  color: #dc3545;
}

.transaction-actions {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.action-icon {
  font-size: 1.25rem;
  cursor: pointer;
  transition: all 0.2s;
}

.edit-icon {
  color: #0d6efd;
}

.edit-icon:hover {
  color: #0a58ca;
  transform: scale(1.1);
}

.delete-icon {
  color: #dc3545;
}

.delete-icon:hover {
  color: #bb2d3b;
  transform: scale(1.1);
}

@media (max-width: 768px) {
  .transaction-card {
    flex-wrap: wrap;
  }

  .transaction-info {
    width: 100%;
    order: 2;
  }

  .transaction-amount {
    order: 1;
    min-width: auto;
  }

  .transaction-actions {
    order: 3;
    width: 100%;
    justify-content: flex-end;
  }

  .transaction-details {
    flex-direction: column;
    gap: 0.25rem;
  }
}
</style>
