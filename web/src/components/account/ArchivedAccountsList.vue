<script setup>
import { DateTime } from 'luxon';
import { Services } from '@/services/servicesConfig';
import AccountTypeBadge from '@/components/account/AccountTypeBadge.vue';

const props = defineProps(['archivedAccounts', 'reReadAllAccounts']);

const formatDate = (datetime) =>
  DateTime.fromISO(datetime).isValid
    ? DateTime.fromISO(datetime).toLocaleString(DateTime.DATETIME_SHORT)
    : '-';

const restoreAccount = async (id) => {
  const updated = await Services.accountsService.setArchivedStatus(id, false);
  if (updated) {
    await props.reReadAllAccounts(true);
  }
};
</script>

<template>
  <main>
    <div class="container">
      <div
        class="row list-item"
        v-for="acc in props.archivedAccounts"
        :key="acc.id">
        <div class="col">
          {{ acc.name }}
          <AccountTypeBadge
            v-if="acc.accountTypeId"
            :type-id="acc.accountTypeId"
            class="ms-1" />
        </div>
        <div class="col">{{ acc.balance }}&nbsp;{{ acc.currency.code }}</div>
        <div class="col-4">
          <b>{{ $t('message.archiveDate') }}</b>
          : {{ formatDate(acc.archivedAt) }}
        </div>
        <div class="col">
          <button
            class="btn btn-primary"
            @click="restoreAccount(acc.id)">
            {{ $t('buttons.restore') }}
          </button>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped lang="scss">
@use '@/assets/main.scss' as *;
</style>
