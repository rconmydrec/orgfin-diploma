<script setup>
import ListContainer from '../components/transactions/ListContainer.vue';

const props = defineProps([
  'accountId',
  'isAccountDetails',
  'categories',
  'startDate',
  'endDate',
]);
</script>

<template>
  <main>
    <ListContainer
      :account-id="props.accountId"
      :initial-categories="props.categories"
      :start-date="props.startDate"
      :end-date="props.endDate"
      :is-account-details="!!props.isAccountDetails" />
  </main>
</template>

<style scoped lang="scss">
@use '@/assets/main.scss' as *;

.transaction-element {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.amount-container {
  text-align: right;
}

.active-filter {
  font-weight: bold;
  color: #ffffff;
}
</style>
