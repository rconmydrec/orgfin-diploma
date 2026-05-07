<script setup>
import { RouterLink } from 'vue-router';
import AccountTypeBadge from '@/components/account/AccountTypeBadge.vue';

const props = defineProps(['accounts', 'totalBalance', 'baseCurrencyCode']);

function balanceClass(balance) {
  return balance < 0 ? 'text-danger' : 'text-success';
}

function availableBalanceCC(acc) {
  return acc.balance + acc.creditLimit;
}
</script>

<template>
  <div class="row">
    <div class="col">
      <div class="accounts-header">
        <b>{{ $t('message.yourAccounts') }}</b>
        ( {{ $t('message.balance') }}: {{ $n(totalBalance, 'decimal') }}
        {{ baseCurrencyCode }})
      </div>
    </div>
  </div>
  <div
    v-for="acc in props.accounts"
    :key="acc.id"
    class="list-item">
    <RouterLink
      class="account-link"
      :to="{ name: 'accountDetails', params: { id: acc.id } }">
      <div class="row account-item">
        <div class="col-7 account-name">
          <span class="name-text">{{ acc.name }}</span>
          <AccountTypeBadge
            :type-id="acc.accountTypeId"
            class="account-type-badge-item" />
        </div>
        <div
          class="col account-balance"
          :class="balanceClass(acc.balance)">
          <div>
            <b>{{ $n(acc.balance, 'decimal') }}</b>
            <span v-if="acc.accountTypeId === 4">
              ({{ $n(availableBalanceCC(acc), 'decimal') }})
            </span>
            {{ acc.currency.code }}
          </div>
          <div v-if="baseCurrencyCode !== acc.currency.code">
            ({{ $n(acc.balanceInBaseCurrency, 'decimal') }}
            {{ baseCurrencyCode }})
          </div>
        </div>
      </div>
    </RouterLink>
  </div>
</template>

<style scoped lang="scss">
@use '@/assets/main.scss' as *;

.accounts-header {
  white-space: nowrap;
  font-size: clamp(0.75rem, 3.5vw, 1rem);
}

.account-name {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  line-height: 1.3;
}

.name-text {
  word-break: break-word;
}

.account-balance {
  text-align: right;
}

.list-item > a {
  text-decoration: none;
  color: black;
}

.account-marks {
  text-align: right;
}

.account-type-badge-item {
  flex-shrink: 0;
}
</style>
