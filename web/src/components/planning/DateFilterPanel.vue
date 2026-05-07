<script setup>
import { computed } from 'vue';

const props = defineProps({
  startDate: {
    type: String,
    default: '',
  },
  endDate: {
    type: String,
    default: '',
  },
});

const emit = defineEmits(['update:startDate', 'update:endDate', 'reset']);

const localStartDate = computed({
  get: () => props.startDate,
  set: (value) => emit('update:startDate', value),
});

const localEndDate = computed({
  get: () => props.endDate,
  set: (value) => emit('update:endDate', value),
});

const isFilterActive = computed(() => {
  return props.startDate || props.endDate;
});

function handleReset() {
  emit('reset');
}
</script>

<template>
  <div class="date-filter mb-4">
    <div class="row g-3 align-items-end">
      <div class="col-md-4">
        <label class="form-label">
          {{ $t('financialPlanning.startDate') }}
        </label>
        <input
          type="date"
          class="form-control"
          v-model="localStartDate" />
      </div>
      <div class="col-md-4">
        <label class="form-label">{{ $t('financialPlanning.endDate') }}</label>
        <input
          type="date"
          class="form-control"
          v-model="localEndDate" />
      </div>
      <div class="col-md-4">
        <button
          class="btn btn-outline-secondary w-100"
          @click="handleReset"
          :disabled="!isFilterActive">
          <i class="bi bi-x-circle"></i>
          {{ $t('financialPlanning.resetFilter') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.date-filter {
  padding: 1rem;
  background: #f8f9fa;
  border-radius: 8px;
}

.date-filter .form-label {
  font-weight: 500;
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
  color: #495057;
}

.date-filter .form-control {
  border-radius: 8px;
}
</style>
