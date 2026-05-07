<template>
  <div class="statistics-panel">
    <div class="stat-card current-balance">
      <div class="stat-icon">
        <i class="bi bi-wallet2"></i>
      </div>
      <div class="stat-content">
        <div class="stat-label">
          {{ $t('financialPlanning.currentBalance') }}
        </div>
        <div class="stat-value">{{ formatCurrency(currentBalance) }}</div>
      </div>
    </div>

    <div class="stat-card projected-balance">
      <div class="stat-icon">
        <i class="bi bi-graph-up-arrow"></i>
      </div>
      <div class="stat-content">
        <div class="stat-label">
          {{ $t('financialPlanning.projectedBalance') }}
        </div>
        <div class="stat-value">{{ formatCurrency(projectedBalance) }}</div>
        <div class="stat-subtitle">
          {{ projectionPeriodLabel }}
        </div>
      </div>
    </div>

    <div class="stat-card income-card">
      <div class="stat-icon">
        <i class="bi bi-arrow-up-circle"></i>
      </div>
      <div class="stat-content">
        <div class="stat-label">{{ $t('financialPlanning.stats.income') }}</div>
        <div class="stat-value text-success">
          {{ formatCurrency(totalIncome) }}
        </div>
        <div class="stat-subtitle">
          {{ incomeCount }} {{ $t('financialPlanning.transactions') }}
        </div>
      </div>
    </div>

    <div class="stat-card expenses-card">
      <div class="stat-icon">
        <i class="bi bi-arrow-down-circle"></i>
      </div>
      <div class="stat-content">
        <div class="stat-label">
          {{ $t('financialPlanning.stats.expenses') }}
        </div>
        <div class="stat-value text-danger">
          {{ formatCurrency(totalExpenses) }}
        </div>
        <div class="stat-subtitle">
          {{ expensesCount }} {{ $t('financialPlanning.transactions') }}
        </div>
      </div>
    </div>

    <div
      class="stat-card difference-card"
      :class="differenceClass">
      <div class="stat-icon">
        <i :class="differenceIcon"></i>
      </div>
      <div class="stat-content">
        <div class="stat-label">{{ $t('financialPlanning.netChange') }}</div>
        <div
          class="stat-value"
          :class="differenceClass">
          {{ formatCurrency(difference) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
  currentBalance: {
    type: Number,
    required: true,
  },
  projectedBalance: {
    type: Number,
    required: true,
  },
  totalIncome: {
    type: Number,
    required: true,
  },
  totalExpenses: {
    type: Number,
    required: true,
  },
  incomeCount: {
    type: Number,
    default: 0,
  },
  expensesCount: {
    type: Number,
    default: 0,
  },
  currency: {
    type: String,
    default: 'USD',
  },
  projectionEndDate: {
    type: String,
    default: null,
  },
});

const difference = computed(() => props.totalIncome - props.totalExpenses);

const differenceClass = computed(() => {
  if (difference.value > 0) return 'text-success';
  if (difference.value < 0) return 'text-danger';
  return 'text-muted';
});

const differenceIcon = computed(() => {
  if (difference.value > 0) return 'bi bi-arrow-up';
  if (difference.value < 0) return 'bi bi-arrow-down';
  return 'bi bi-dash';
});

const projectionPeriodLabel = computed(() => {
  if (!props.projectionEndDate) {
    return t('financialPlanning.in30Days');
  }

  const endDate = new Date(props.projectionEndDate);
  const today = new Date();
  const diffTime = endDate - today;
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays <= 0) {
    return t('financialPlanning.today');
  } else if (diffDays === 1) {
    return t('financialPlanning.tomorrow');
  } else if (diffDays <= 7) {
    return t('financialPlanning.inNDays', { n: diffDays });
  } else if (diffDays <= 30) {
    return t('financialPlanning.inNDays', { n: diffDays });
  } else if (diffDays <= 365) {
    const months = Math.floor(diffDays / 30);
    return t('financialPlanning.inNMonths', { n: months });
  } else {
    const years = Math.floor(diffDays / 365);
    return t('financialPlanning.inNYears', { n: years });
  }
});

function formatCurrency(amount) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: props.currency,
  }).format(amount);
}
</script>

<style scoped>
.statistics-panel {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 1rem;
  transition:
    transform 0.2s,
    box-shadow 0.2s;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.stat-icon {
  font-size: 2.5rem;
  color: #6c757d;
  opacity: 0.7;
}

.current-balance .stat-icon {
  color: #0d6efd;
}

.projected-balance .stat-icon {
  color: #6610f2;
}

.income-card .stat-icon {
  color: #198754;
}

.expenses-card .stat-icon {
  color: #dc3545;
}

.difference-card .stat-icon {
  color: #ffc107;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 0.875rem;
  color: #6c757d;
  margin-bottom: 0.25rem;
  font-weight: 500;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #212529;
  margin-bottom: 0.25rem;
}

.stat-subtitle {
  font-size: 0.75rem;
  color: #6c757d;
}

@media (max-width: 768px) {
  .statistics-panel {
    grid-template-columns: 1fr;
  }

  .stat-card {
    padding: 1rem;
  }

  .stat-value {
    font-size: 1.25rem;
  }

  .stat-icon {
    font-size: 2rem;
  }
}
</style>
