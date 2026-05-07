<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
  limitType: {
    type: String,
    required: true,
    validator: (value) => ['accounts', 'budgets', 'planning'].includes(value),
  },
  limit: {
    type: Number,
    required: true,
  },
  show: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits(['upgrade', 'close']);

const alertMessage = computed(() => {
  switch (props.limitType) {
    case 'accounts':
      return t('subscription.accountLimitReached', { limit: props.limit });
    case 'budgets':
      return t('subscription.budgetLimitReached', { limit: props.limit });
    case 'planning':
      return t('subscription.planningLimitReached', { days: props.limit });
    default:
      return t('subscription.limitReached');
  }
});

const alertIcon = computed(() => {
  switch (props.limitType) {
    case 'accounts':
      return 'fa-wallet';
    case 'budgets':
      return 'fa-chart-pie';
    case 'planning':
      return 'fa-calendar-alt';
    default:
      return 'fa-exclamation-circle';
  }
});

function handleUpgrade() {
  emit('upgrade');
}

function handleClose() {
  emit('close');
}
</script>

<template>
  <div
    v-if="show"
    class="alert alert-warning alert-dismissible fade show limit-alert"
    role="alert">
    <div class="alert-content">
      <div class="alert-icon">
        <i
          class="fa-solid"
          :class="alertIcon"></i>
      </div>
      <div class="alert-text">
        <strong>{{ t('subscription.limitReachedTitle') }}</strong>
        <p class="mb-0">{{ alertMessage }}</p>
      </div>
    </div>
    <div class="alert-actions">
      <button
        class="btn btn-sm btn-warning"
        @click="handleUpgrade">
        <i class="fa-solid fa-crown me-1"></i>
        {{ t('subscription.upgradeToPremium') }}
      </button>
      <button
        type="button"
        class="btn-close"
        @click="handleClose"
        :aria-label="t('buttons.close')"></button>
    </div>
  </div>
</template>

<style scoped>
.limit-alert {
  border-left: 4px solid #ffc107;
  background-color: #fff3cd;
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.alert-content {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  flex: 1;
}

.alert-icon {
  font-size: 1.5rem;
  color: #856404;
  flex-shrink: 0;
}

.alert-text {
  flex: 1;
}

.alert-text strong {
  display: block;
  margin-bottom: 0.25rem;
  color: #856404;
}

.alert-text p {
  color: #856404;
  font-size: 0.9rem;
}

.alert-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-warning {
  color: #fff;
  background-color: #ffc107;
  border-color: #ffc107;
  font-weight: 600;
}

.btn-warning:hover {
  background-color: #e0a800;
  border-color: #d39e00;
}

.btn-close {
  padding: 0.5rem;
}

@media (max-width: 768px) {
  .limit-alert {
    flex-direction: column;
    align-items: flex-start;
  }

  .alert-actions {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
