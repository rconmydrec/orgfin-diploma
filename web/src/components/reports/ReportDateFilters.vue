<script setup>
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
  startDate: {
    type: String,
    required: true,
  },
  endDate: {
    type: String,
    required: true,
  },
  startDateLabel: {
    type: String,
    default: 'message.startDate',
  },
  endDateLabel: {
    type: String,
    default: 'message.endDate',
  },
  showPeriod: {
    type: Boolean,
    default: false,
  },
  period: {
    type: String,
    default: 'monthly',
  },
  periodOptions: {
    type: Array,
    default: () => [
      { value: 'daily', label: 'message.daily' },
      { value: 'monthly', label: 'message.monthly' },
    ],
  },
});

const emit = defineEmits([
  'update:startDate',
  'update:endDate',
  'update:period',
  'change',
]);

function onStartDateChange(event) {
  emit('update:startDate', event.target.value);
  emit('change');
}

function onEndDateChange(event) {
  emit('update:endDate', event.target.value);
  emit('change');
}

function onPeriodChange(event) {
  emit('update:period', event.target.value);
  emit('change');
}
</script>

<template>
  <div class="filters-section">
    <label>
      {{ t(startDateLabel) }}
      <input
        type="date"
        class="form-control"
        :value="startDate"
        @change="onStartDateChange" />
    </label>

    <label>
      {{ t(endDateLabel) }}
      <input
        type="date"
        class="form-control"
        :value="endDate"
        @change="onEndDateChange" />
    </label>

    <label v-if="showPeriod">
      {{ t('message.period') }}
      <select
        class="form-control"
        :value="period"
        @change="onPeriodChange">
        <option
          v-for="opt in periodOptions"
          :key="opt.value"
          :value="opt.value">
          {{ t(opt.label) }}
        </option>
      </select>
    </label>

    <slot></slot>
  </div>
</template>

<style scoped>
.filters-section {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  margin-top: 16px;
  align-items: end;
}

@media (max-width: 767px) {
  .filters-section {
    grid-template-columns: 1fr;
  }
}
</style>
