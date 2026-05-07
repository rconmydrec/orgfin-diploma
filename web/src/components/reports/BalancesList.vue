<script setup>
import { useI18n } from 'vue-i18n';
import AccountTypeBadge from '@/components/account/AccountTypeBadge.vue';

defineProps(['balanceData', 'totalBalance', 'baseCurrencyCode']);

const t = useI18n().t;

const baseBalanceClass = (balance) => {
  if (balance >= 0) {
    return 'positive-balance';
  } else {
    return 'negative-balance';
  }
};

const accountName = (name) => {
  if (name.length === 0) {
    return `(${t('message.didntExist')})`;
  } else {
    return name;
  }
};
</script>

<template>
  <div
    class="account-item"
    v-for="balance in balanceData"
    :key="balance.id">
    <div class="account-name">
      <AccountTypeBadge
        v-if="balance.accountTypeId"
        :type-id="balance.accountTypeId"
        class="me-1" />
      {{ accountName(balance.accountName) }}
    </div>
    <div
      class="account-balance-base"
      :class="baseBalanceClass(balance.baseCurrencyBalance)">
      {{ $n(balance.baseCurrencyBalance, 'decimal') }}
      {{ balance.baseCurrencyCode }}
    </div>
    <div class="account-balance-original">
      {{ $n(balance.balance, 'decimal') }} {{ balance.currencyCode }}
    </div>
  </div>

  <hr />

  <div class="account-total-balance">
    <div class="account-name">{{ $t('message.balance') }}</div>
    <div
      class="account-balance-base"
      :class="baseBalanceClass(totalBalance)">
      {{ $n(totalBalance, 'decimal') }} {{ baseCurrencyCode }}
    </div>
  </div>
</template>

<style scoped lang="scss">
@use '@/assets/common.scss' as *;

.account-total-balance {
  display: grid;
  grid-template-areas:
    'account-name account-name'
    'account-balance-base account-balance-base';
  grid-template-columns: 1fr 1fr;
  background-color: #516fa2;
  padding: 5px;
  margin-bottom: 10px;
  border-radius: 5px;
}

.account-item {
  display: grid;
  grid-template-areas:
    'account-name account-balance-base'
    'account-balance-original account-balance-original';
  grid-template-columns: 1fr 1fr;
  grid-template-rows: auto auto;
  background-color: $item-background-color;
  padding: 5px;
  margin-bottom: 10px;
  border-radius: 5px;
}

.account-name {
  grid-area: account-name;
}

.account-balance-base {
  grid-area: account-balance-base;
  justify-self: end;
  font-weight: bold;
  font-size: 1.2rem;
}

.positive-balance {
  color: #81de5f;
}

.negative-balance {
  color: #781919;
}

.account-balance-original {
  grid-area: account-balance-original;
  justify-self: end;
}

@media (max-width: 768px) {
  .account-item {
    grid-template-areas:
      'account-name'
      'account-balance-base'
      'account-balance-original';
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
  }

  .account-balance-base {
    font-size: 1rem;
  }
}
</style>
