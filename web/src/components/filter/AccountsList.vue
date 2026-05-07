<script setup>
import { onBeforeMount, reactive, watch } from 'vue';
import { useRouter } from 'vue-router';

import { processError } from '../../errors/errorHandlers';
import { Services } from '../../services/servicesConfig';

const props = defineProps(['selectedAccounts']);
const emit = defineEmits(['selectedAccountsUpdated']);
const router = useRouter();

const accounts = reactive([]);

const selectedAccs = reactive([]);
watch(props['selectedAccounts'], (newSelectedAccounts) => {
  if (newSelectedAccounts.length === 0) {
    selectedAccs.length = 0;
  }
});

onBeforeMount(async () => {
  try {
    accounts.length = 0;
    const tmpAccounts = await Services.accountsService.getUserAccounts({
      shouldUpdate: true,
    });
    if (tmpAccounts) {
      accounts.push(...tmpAccounts);
    }
  } catch (e) {
    await processError(e, router);
  }
});

function toggleAccountInFilter(event) {
  const accId = parseInt(event.target.value, 10);
  const idxInSelected = selectedAccs.indexOf(accId);
  if (idxInSelected === -1) {
    selectedAccs.push(accId);
  } else {
    selectedAccs.splice(idxInSelected, 1);
  }
  emit('selectedAccountsUpdated', {
    selectedAccounts: selectedAccs,
  });
}
</script>

<template>
  <div class="row">
    <div class="col">
      <div
        class="list-item account-item"
        v-for="acc in accounts"
        :key="acc.id">
        <span class="acc-name">
          <input
            class="form-check-input"
            type="checkbox"
            :value="acc.id"
            @click="toggleAccountInFilter"
            :checked="props.selectedAccounts.includes(parseInt(acc.id, 10))" />
          <label
            class="form-check-label"
            for="form-check-input">
            {{ acc.name }}
          </label>
        </span>
        <span>( {{ acc.currency.code }} {{ acc.balance }} )</span>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@use '@/assets/common.scss' as *;

.account-item {
  display: flex;
  justify-content: space-between;
  background-color: $item-background-color;
}

.account-item:hover {
  background-color: $item-hover-background-color;
}

.form-check-input {
  margin-right: 1vw;
}

.row {
  max-height: 10rem;
  overflow-y: auto;
}
</style>
