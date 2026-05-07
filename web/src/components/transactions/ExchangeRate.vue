<script setup>
import { computed } from 'vue';

const props = defineProps([
  'srcAmount',
  'targetAmount',
  'currencySrcCode',
  'currencyTargetCode',
]);

const exchangeRate = computed(() => {
  return props.targetAmount / props.srcAmount;
});

const reversedExchangedRate = computed(() => {
  return props.srcAmount / props.targetAmount;
});
</script>

<template>
  <div>
    {{ currencySrcCode }} / {{ currencyTargetCode }}:
    {{ exchangeRate.toFixed(4) }}
  </div>
  <div class="mb-2">
    {{ currencyTargetCode }} / {{ currencySrcCode }}: ({{
      reversedExchangedRate.toFixed(4)
    }})
  </div>
</template>

<style scoped></style>
