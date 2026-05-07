<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import CalculatorInput from '@/components/utilities/CalculatorInput.vue';

const { t } = useI18n();

const props = defineProps({
  modelValue: {
    type: Boolean,
    required: true,
  },
  transaction: {
    type: Object,
    default: null,
  },
  loading: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['update:modelValue', 'submit']);

const isEdit = computed(() => !!props.transaction?.id);

const endConditionType = ref('endDate');

const defaultForm = {
  amount: 0,
  label: '',
  notes: '',
  isIncome: false,
  plannedDate: '',
  isRecurring: false,
  isActive: true,
  recurrenceRule: {
    frequency: 'monthly',
    interval: 1,
    endDate: null,
    count: null,
  },
};

const amountInput = ref('');

const form = ref({ ...defaultForm });

// Handle amount input
function handleAmountInput(newValue) {
  amountInput.value = newValue;

  const parsed = parseFloat(newValue);
  if (!isNaN(parsed)) {
    form.value.amount = parsed;
  }
}

// Watch for transaction prop changes (for edit mode)
watch(
  () => props.transaction,
  (newTransaction) => {
    if (newTransaction) {
      form.value = {
        id: newTransaction.id,
        amount: newTransaction.amount,
        label: newTransaction.label,
        notes: newTransaction.notes || '',
        isIncome: newTransaction.isIncome,
        plannedDate: newTransaction.plannedDate?.split('T')[0] || '',
        isRecurring: newTransaction.isRecurring,
        isActive:
          newTransaction.isActive !== undefined
            ? newTransaction.isActive
            : true,
        recurrenceRule: {
          frequency: newTransaction.recurrenceRule?.frequency || 'monthly',
          interval: newTransaction.recurrenceRule?.interval || 1,
          endDate:
            newTransaction.recurrenceRule?.endDate?.split('T')[0] || null,
          count: newTransaction.recurrenceRule?.count || null,
        },
      };
      amountInput.value = String(newTransaction.amount ?? '');

      // Set end condition type
      if (newTransaction.recurrenceRule?.endDate) {
        endConditionType.value = 'endDate';
      } else if (newTransaction.recurrenceRule?.count) {
        endConditionType.value = 'count';
      }
    } else {
      resetForm();
    }
  },
  { immediate: true },
);

function resetForm() {
  form.value = { ...defaultForm };
  amountInput.value = '';
  endConditionType.value = 'endDate';
}

function closeModal() {
  emit('update:modelValue', false);
  setTimeout(resetForm, 300); // Wait for modal animation
}

function handleSubmit() {
  // Prepare recurrence rule
  const recurrenceRule = form.value.isRecurring
    ? {
        frequency: form.value.recurrenceRule.frequency,
        interval: form.value.recurrenceRule.interval,
        endDate:
          endConditionType.value === 'endDate'
            ? form.value.recurrenceRule.endDate
            : null,
        count:
          endConditionType.value === 'count'
            ? form.value.recurrenceRule.count
            : null,
      }
    : null;

  const data = {
    ...form.value,
    recurrenceRule,
  };

  emit('submit', data);
}

function getIntervalHint() {
  const freq = form.value.recurrenceRule.frequency;
  const interval = form.value.recurrenceRule.interval;

  const hints = {
    daily: t('financialPlanning.everyNDays', { n: interval }),
    weekly: t('financialPlanning.everyNWeeks', { n: interval }),
    monthly: t('financialPlanning.everyNMonths', { n: interval }),
    yearly: t('financialPlanning.everyNYears', { n: interval }),
  };

  return hints[freq] || '';
}
</script>

<template>
  <div
    class="modal fade"
    :class="{ show: modelValue }"
    :style="{ display: modelValue ? 'block' : 'none' }"
    tabindex="-1"
    @click.self="closeModal">
    <div class="modal-dialog modal-lg">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">
            {{
              isEdit
                ? $t('financialPlanning.editPlanned')
                : $t('financialPlanning.createPlanned')
            }}
          </h5>
          <button
            type="button"
            class="btn-close"
            @click="closeModal"></button>
        </div>

        <form @submit.prevent="handleSubmit">
          <div class="modal-body">
            <!-- Transaction Type -->
            <div class="mb-3">
              <label class="form-label">
                {{ $t('financialPlanning.transactionType') }}
              </label>
              <div
                class="btn-group w-100"
                role="group">
                <input
                  type="radio"
                  class="btn-check"
                  id="type-expense"
                  v-model="form.isIncome"
                  :value="false"
                  autocomplete="off" />
                <label
                  class="btn btn-outline-danger"
                  for="type-expense">
                  <i class="bi bi-arrow-down-circle"></i>
                  {{ $t('message.expense') }}
                </label>

                <input
                  type="radio"
                  class="btn-check"
                  id="type-income"
                  v-model="form.isIncome"
                  :value="true"
                  autocomplete="off" />
                <label
                  class="btn btn-outline-success"
                  for="type-income">
                  <i class="bi bi-arrow-up-circle"></i>
                  {{ $t('message.income') }}
                </label>
              </div>
            </div>

            <!-- Amount -->
            <div class="mb-3">
              <label class="form-label">
                {{ $t('message.amount') }}
                <span class="text-danger">*</span>
              </label>
              <CalculatorInput
                v-model="amountInput"
                @update:modelValue="handleAmountInput"
                :placeholder="'0.00'"
                :min="0" />
            </div>

            <!-- Label -->
            <div class="mb-3">
              <label class="form-label">
                {{ $t('message.label') }}
                <span class="text-danger">*</span>
              </label>
              <input
                v-model="form.label"
                type="text"
                class="form-control"
                maxlength="50"
                required />
            </div>

            <!-- Notes -->
            <div class="mb-3">
              <label class="form-label">{{ $t('message.notes') }}</label>
              <textarea
                v-model="form.notes"
                class="form-control"
                rows="2"></textarea>
            </div>

            <!-- Planned Date -->
            <div class="mb-3">
              <label class="form-label">
                {{ $t('financialPlanning.form.plannedDate') }}
                <span class="text-danger">*</span>
              </label>
              <input
                v-model="form.plannedDate"
                type="date"
                class="form-control"
                required />
            </div>

            <!-- Is Recurring Toggle -->
            <div class="mb-3">
              <div class="form-check form-switch">
                <input
                  v-model="form.isRecurring"
                  class="form-check-input"
                  type="checkbox"
                  id="isRecurring" />
                <label
                  class="form-check-label"
                  for="isRecurring">
                  <i class="bi bi-arrow-repeat"></i>
                  {{ $t('financialPlanning.form.isRecurring') }}
                </label>
              </div>
            </div>

            <!-- Is Active Toggle -->
            <div class="mb-3">
              <div class="form-check form-switch">
                <input
                  v-model="form.isActive"
                  class="form-check-input"
                  type="checkbox"
                  id="isActive" />
                <label
                  class="form-check-label"
                  for="isActive">
                  <i class="bi bi-check-circle"></i>
                  {{ $t('financialPlanning.form.isActive') }}
                </label>
              </div>
            </div>

            <!-- Recurrence Settings (shown only if recurring) -->
            <div
              v-if="form.isRecurring"
              class="card bg-light">
              <div class="card-body">
                <h6 class="card-title">
                  {{ $t('financialPlanning.recurrenceSettings') }}
                </h6>

                <!-- Frequency -->
                <div class="mb-3">
                  <label class="form-label">
                    {{ $t('financialPlanning.form.frequency') }}
                  </label>
                  <select
                    v-model="form.recurrenceRule.frequency"
                    class="form-select">
                    <option value="daily">
                      {{ $t('financialPlanning.recurrenceFrequency.daily') }}
                    </option>
                    <option value="weekly">
                      {{ $t('financialPlanning.recurrenceFrequency.weekly') }}
                    </option>
                    <option value="monthly">
                      {{ $t('financialPlanning.recurrenceFrequency.monthly') }}
                    </option>
                    <option value="yearly">
                      {{ $t('financialPlanning.recurrenceFrequency.yearly') }}
                    </option>
                  </select>
                </div>

                <!-- Interval -->
                <div class="mb-3">
                  <label class="form-label">
                    {{ $t('financialPlanning.form.interval') }}
                  </label>
                  <input
                    v-model.number="form.recurrenceRule.interval"
                    type="number"
                    min="1"
                    class="form-control" />
                  <small class="text-muted">
                    {{ getIntervalHint() }}
                  </small>
                </div>

                <!-- End condition -->
                <div class="mb-3">
                  <label class="form-label">
                    {{ $t('financialPlanning.endCondition') }}
                  </label>
                  <div
                    class="btn-group w-100"
                    role="group">
                    <input
                      type="radio"
                      class="btn-check"
                      id="end-date"
                      v-model="endConditionType"
                      value="endDate"
                      autocomplete="off" />
                    <label
                      class="btn btn-outline-primary"
                      for="end-date">
                      {{ $t('financialPlanning.form.endDate') }}
                    </label>

                    <input
                      type="radio"
                      class="btn-check"
                      id="end-count"
                      v-model="endConditionType"
                      value="count"
                      autocomplete="off" />
                    <label
                      class="btn btn-outline-primary"
                      for="end-count">
                      {{ $t('financialPlanning.form.count') }}
                    </label>
                  </div>
                </div>

                <!-- End Date (if selected) -->
                <div
                  v-if="endConditionType === 'endDate'"
                  class="mb-3">
                  <input
                    v-model="form.recurrenceRule.endDate"
                    type="date"
                    class="form-control" />
                </div>

                <!-- Count (if selected) -->
                <div
                  v-if="endConditionType === 'count'"
                  class="mb-3">
                  <input
                    v-model.number="form.recurrenceRule.count"
                    type="number"
                    min="1"
                    class="form-control"
                    :placeholder="
                      $t('financialPlanning.numberOfOccurrences')
                    " />
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="closeModal">
              {{ $t('common.cancel') }}
            </button>
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="loading">
              <span
                v-if="loading"
                class="spinner-border spinner-border-sm me-2"></span>
              {{ isEdit ? $t('common.update') : $t('common.create') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
  <div
    v-if="modelValue"
    class="modal-backdrop fade show"></div>
</template>

<style scoped>
.modal.show {
  display: block !important;
}

.modal-backdrop {
  background-color: rgba(0, 0, 0, 0.5);
}

.btn-check:checked + .btn-outline-danger {
  background-color: #dc3545;
  color: white;
}

.btn-check:checked + .btn-outline-success {
  background-color: #198754;
  color: white;
}

.btn-check:checked + .btn-outline-primary {
  background-color: #0d6efd;
  color: white;
}

.form-check-input:checked {
  background-color: #0d6efd;
  border-color: #0d6efd;
}
</style>
