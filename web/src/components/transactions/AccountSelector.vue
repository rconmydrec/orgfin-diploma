<script setup>
import { computed } from 'vue';

const props = defineProps(['accounts', 'label', 'accountType', 'accountId']);
const emit = defineEmits(['accountChanged']);

function changeAccount($event) {
  const acc = props.accounts[$event.target.value];
  emit('accountChanged', {
    accountType: props.accountType,
    accountId: acc.id,
  });
}

const accountIdx = computed(() => {
  return props.accounts.findIndex((acc) => acc.id === props.accountId);
});
</script>

<template>
  <label
    for="label"
    class="form-label">
    {{ label }}
  </label>
  <select
    class="form-select bottom-space"
    @change="changeAccount"
    :value="accountIdx">
    <option
      v-for="(acc, index) in accounts"
      :key="acc.id"
      :value="index">
      {{ acc.name }} ({{ $n(acc.balance) }} {{ acc.currency.code }})
    </option>
  </select>
</template>
