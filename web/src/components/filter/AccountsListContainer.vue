<script setup>
import AccountsList from '@/components/filter/AccountsList.vue';
import CustomHr from '@/components/utilities/CustomHr.vue';

defineProps(['selectedAccounts', 'selectedAccountsUpdated']);
const emit = defineEmits(['selectedAccountsUpdated']);

function handleSelectedAccountsUpdated({ selectedAccounts }) {
  emit('selectedAccountsUpdated', {
    selectedAccounts: selectedAccounts,
  });
}
</script>

<template>
  <div>
    <div class="row">
      <div class="col">
        <CustomHr text="Accounts" />
      </div>
    </div>

    <div class="row">
      <div class="col">
        <AccountsList
          @selected-accounts-updated="handleSelectedAccountsUpdated"
          :selected-accounts="selectedAccounts" />
      </div>
    </div>
  </div>
</template>

<style scoped></style>
