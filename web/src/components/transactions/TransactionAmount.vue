<script setup>
import { ref, defineExpose, watch } from 'vue';
import CalculatorInput from '@/components/utilities/CalculatorInput.vue';

const props = defineProps(['amount', 'currentAccount', 'label', 'type']);
const emit = defineEmits(['amountChanged']);

const amountInput = ref(null);
defineExpose({ amountInput });

const inputValue = ref(props.amount ?? '');

watch(
  () => props.amount,
  (newVal) => {
    inputValue.value = newVal ?? '';
  },
  { immediate: true },
);

function changeAmount(newValue) {
  inputValue.value = newValue;

  const parsed = parseFloat(newValue);
  if (!isNaN(parsed)) {
    emit('amountChanged', {
      amountType: props.type,
      amount: parsed,
    });
  }
}
</script>

<template>
  <div class="mb-3">
    <div class="row">
      <div class="col-10">
        <label
          for="amount"
          class="form-label">
          {{ label }}
        </label>
        <CalculatorInput
          ref="amountInput"
          v-model="inputValue"
          @update:modelValue="changeAmount"
          :placeholder="'0.00'"
          :min="0" />
      </div>
      <div class="col-2 currency">{{ currentAccount?.currency?.code }}</div>
    </div>
  </div>
</template>

<style scoped>
.currency {
  align-self: flex-end;
}
</style>
