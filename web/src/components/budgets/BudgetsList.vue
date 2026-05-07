<script setup>
const props = defineProps({
  budgets: {
    type: Array,
    required: true,
  },
});

const emit = defineEmits(['budgetSelected']);

const formatDate = (date) => {
  return new Date(date).toLocaleDateString();
};

const budgetSelected = (budget) => {
  emit('budgetSelected', budget);
};

const collectedAmountClass = (budget) => {
  const collectedAmountPercentage =
    (budget.collectedAmount / budget.targetAmount) * 100;
  let commonClass = '';
  if (collectedAmountPercentage < 70) {
    return commonClass + ' low-risk';
  } else if (collectedAmountPercentage < 100) {
    return commonClass + ' medium-risk';
  } else {
    return commonClass + ' high-risk';
  }
};
</script>

<template>
  <div>
    <div class="budget-item-container headers">
      <span class="col-4">{{ $t('message.name') }}</span>
      <span class="col-1">{{ $t('message.collectedExpenses') }}</span>
      <span class="col-1">{{ $t('message.targetBudget') }}</span>
      <span class="col-1">{{ $t('message.budgetLeft') }}</span>
      <span class="col-1">{{ $t('message.startDate') }}</span>
      <span class="col-1">{{ $t('message.endDate') }}</span>
      <span class="col-1">{{ $t('message.active') }}</span>
    </div>
    <div
      v-for="budget in props.budgets"
      :key="budget.id">
      <div
        class="budget-item-container"
        :class="{ 'active-budget': !budget.isArchived }"
        @click="budgetSelected(budget)">
        <span class="data-cell col-4">
          {{ budget.name }} ({{ budget.currency.code }})
        </span>
        <span
          class="data-cell col-1 amount-cell"
          :class="collectedAmountClass(budget)">
          {{ parseInt(budget.collectedAmount, 10) }}
        </span>
        <span class="data-cell col-1 amount-cell">
          {{ parseInt(budget.targetAmount, 10) }}
        </span>
        <span
          class="data-cell col-1 amount-cell"
          :class="collectedAmountClass(budget)">
          {{
            parseInt(budget.targetAmount, 10) -
            parseInt(budget.collectedAmount, 10)
          }}
        </span>
        <span class="data-cell col-1 date-cell">
          {{ formatDate(budget.startDate) }}
        </span>
        <span class="data-cell col-1 date-cell">
          {{ formatDate(budget.endDate) }}
        </span>
        <span class="data-cell col-1 status-cell">
          <span
            class="active"
            v-if="!budget.isArchived">
            &#x2713;
          </span>
          <span
            class="archived"
            v-else>
            &#x2718;
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
@use '@/assets/common.scss' as *;

.budget-item-container {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem;
  border-radius: 0.5rem;
  margin: 0.5rem 0;
  background-color: $item-background-color;
}

.active-budget {
  background-color: $active-item-background-color;
}

.active-budget:hover {
  background-color: $active-item-hover-background-color !important;
}

.budget-item-container:hover {
  background-color: $item-hover-background-color;
  cursor: pointer;
}

.headers {
  font-weight: bold;
  color: #0000ff;
  background-color: #f0f0f0;
  text-align: center;
}

.amount-cell {
  text-align: right;
  font-weight: bold;
}

.low-risk {
  color: rgb(38, 113, 10);
}

.medium-risk {
  color: orange;
}

.high-risk {
  color: #781919;
}

span.active {
  color: green !important;
  font-weight: bold;
}

span.archived {
  color: red !important;
}

.period-cell,
.status-cell,
.date-cell {
  text-align: center;
}

@media (max-width: 768px) {
  .headers span,
  .date-cell,
  .period-cell {
    writing-mode: vertical-lr;
    white-space: nowrap;
  }

  .period-cell {
    flex-direction: column;
    align-items: center;
  }

  .budget-item-container span {
    text-align: left;
  }

  .budget-item-container span:not(:last-child) {
    margin-bottom: 0.5rem;
  }
}

@media (max-width: 480px) {
  .data-cell {
    writing-mode: vertical-lr;
    text-orientation: mixed;
    white-space: nowrap;
  }
}
</style>
