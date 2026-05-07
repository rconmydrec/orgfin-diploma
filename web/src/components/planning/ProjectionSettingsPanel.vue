<script setup>
import { computed } from 'vue';

const props = defineProps({
  endDate: {
    type: String,
    required: true,
  },
  period: {
    type: String,
    required: true,
    validator: (value) => ['daily', 'weekly', 'monthly'].includes(value),
  },
  showInactive: {
    type: Boolean,
    default: false,
  },
  showHidden: {
    type: Boolean,
    default: false,
  },
  inactiveCount: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits([
  'update:endDate',
  'update:period',
  'update:showInactive',
  'update:showHidden',
  'reset',
]);

const localEndDate = computed({
  get: () => props.endDate,
  set: (value) => emit('update:endDate', value),
});

const localPeriod = computed({
  get: () => props.period,
  set: (value) => emit('update:period', value),
});

const localShowInactive = computed({
  get: () => props.showInactive,
  set: (value) => emit('update:showInactive', value),
});

const localShowHidden = computed({
  get: () => props.showHidden,
  set: (value) => emit('update:showHidden', value),
});

function handleReset() {
  emit('reset');
}
</script>

<template>
  <div class="card settings-card">
    <div class="card-body">
      <div class="row align-items-end g-3">
        <div class="col-md-3">
          <label class="form-label">
            {{ $t('financialPlanning.projectionEndDate') }}
          </label>
          <input
            type="date"
            class="form-control"
            v-model="localEndDate" />
        </div>
        <div class="col-md-3">
          <label class="form-label">
            {{ $t('financialPlanning.projectionPeriod') }}
          </label>
          <select
            class="form-select"
            v-model="localPeriod">
            <option value="daily">
              {{ $t('financialPlanning.periods.daily') }}
            </option>
            <option value="weekly">
              {{ $t('financialPlanning.periods.weekly') }}
            </option>
            <option value="monthly">
              {{ $t('financialPlanning.periods.monthly') }}
            </option>
          </select>
        </div>
        <div class="col-md-3">
          <div class="form-check">
            <input
              class="form-check-input"
              type="checkbox"
              id="showInactiveCheckbox"
              v-model="localShowInactive" />
            <label
              class="form-check-label"
              for="showInactiveCheckbox">
              {{ $t('financialPlanning.showInactive') }}
              <span
                v-if="inactiveCount > 0"
                class="badge bg-secondary ms-1">
                {{ inactiveCount }}
              </span>
            </label>
          </div>
          <div class="form-check mt-2">
            <input
              class="form-check-input"
              type="checkbox"
              id="showHiddenCheckbox"
              v-model="localShowHidden" />
            <label
              class="form-check-label"
              for="showHiddenCheckbox">
              {{ $t('financialPlanning.showHidden') }}
            </label>
          </div>
        </div>
        <div class="col-md-3">
          <button
            class="btn btn-outline-secondary w-100"
            @click="handleReset">
            <i class="bi bi-arrow-clockwise"></i>
            {{ $t('financialPlanning.resetSettings') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-card .card-body {
  padding: 1rem 1.5rem;
}

.settings-card .form-label {
  font-weight: 500;
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
  color: #495057;
}

.settings-card .form-control,
.settings-card .form-select {
  border-radius: 8px;
}
</style>
