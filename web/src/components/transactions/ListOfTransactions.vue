<script setup>
import { computed, onBeforeMount, onUnmounted, reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRouter } from 'vue-router';

import { Services } from '../../services/servicesConfig';
import { processError } from '../../errors/errorHandlers';

const props = defineProps(['transactions', 'accountId', 'returnUrl']);
const emit = defineEmits(['loadMore']);

const router = useRouter();
const t = useI18n().t;

const categories = reactive([]);

onBeforeMount(async () => {
  await fetchCategories();
});

async function fetchCategories() {
  try {
    const tmpCategories = await Services.categoriesService.getUserCategories();
    if (tmpCategories) {
      categories.push(...tmpCategories);
    }
  } catch (e) {
    await processError(e, router);
  }
}

async function loadMoreTransactions() {
  emit('loadMore');
}

const groupedTransactions = computed(() => {
  const grouped = [];
  let lastDate = null;

  if (props.transactions) {
    props.transactions.forEach((transaction) => {
      const transactionDate = transaction.dateTime.split('T')[0];
      if (lastDate !== transactionDate) {
        grouped.push({ date: transactionDate, transactions: [] });
        lastDate = transactionDate;
      }
      grouped[grouped.length - 1].transactions.push(transaction);
    });
  }

  return grouped;
});

const getParentCategoryLabel = (transaction) => {
  if (transaction.category) {
    const parentCategory = categories.find(
      (category) => category.id === transaction.category.parentId,
    );
    return parentCategory ? `[${parentCategory.name}] : ` : '';
  }
  return '';
};

const categoryLabel = (transaction) => {
  if (transaction.isAdjustment) {
    return t('message.balanceAdjustment');
  }
  if (transaction.isTransfer) {
    return 'Transfer';
  }
  return transaction.category?.name || 'Unknown';
};

const transactionClass = (transaction) => {
  if (transaction.isAdjustment) {
    return 'adjustment-transaction';
  }
  if (transaction.isIncome) {
    return 'income-transaction';
  } else {
    return 'expense-transaction';
  }
};

const balanceClass = (transaction) => {
  if (transaction.newBalance !== null) {
    return transaction.newBalance >= 0
      ? 'income-transaction'
      : 'expense-transaction';
  }
  return '';
};

function accountName(account) {
  if (account.isDeleted) {
    return `${account.name} (${t('message.deleted')})`;
  } else {
    return account.name;
  }
}

function handleScroll(event) {
  const { scrollTop, clientHeight, scrollHeight } =
    event.target.scrollingElement;
  if (scrollTop + clientHeight >= scrollHeight - 200) {
    loadMoreTransactions();
  }
}

window.addEventListener('scroll', handleScroll);

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll);
});
</script>

<template>
  <div v-if="groupedTransactions.length > 0">
    <div
      v-for="group in groupedTransactions"
      :key="group.date">
      <div class="date-header">{{ group.date }}</div>
      <div
        v-for="transaction in group.transactions"
        :key="transaction.id"
        class="list-item">
        <RouterLink
          class="row"
          :to="{
            name: 'transactionDetails',
            params: {
              id: transaction.id,
            },
            query: {
              returnUrl: props.returnUrl,
              accountId: props.accountId,
            },
          }">
          <div class="col-5">
            <div class="transaction-element">
              <b>{{ transaction.label }}</b>
            </div>
            <div class="transaction-element">
              <b>{{ getParentCategoryLabel(transaction) }}</b>
              {{ categoryLabel(transaction) }}
            </div>
          </div>
          <div class="col-7 amount-container">
            <div :class="transactionClass(transaction)">
              <b>
                {{ $n(transaction.amount, 'decimal')
                }}{{ transaction.account.currency.code }}
              </b>
            </div>
            <div
              v-if="
                transaction.account.currency.code !==
                transaction.baseCurrencyCode
              ">
              ({{ $n(transaction.baseCurrencyAmount, 'decimal') }}
              {{ transaction.baseCurrencyCode }})
            </div>
            <div :class="balanceClass(transaction)">
              <span class="acc-name">
                {{ accountName(transaction.account) }}
              </span>
              |
              <span v-if="transaction.newBalance !== null">
                {{ $n(transaction.newBalance, 'decimal') }}
              </span>
              <span v-else>--</span>
            </div>
          </div>
        </RouterLink>
      </div>
    </div>
  </div>
  <div v-else>No transactions found</div>
</template>

<style scoped>
@use '@/assets/main.scss' as *;

.date-header {
  font-weight: bold;
  margin: 10px 0;
  background-color: #b3b2b2;
  padding: 5px;
  border-radius: 5px;
}

.list-item {
  padding: 5px;
  border: 1px solid #eee;
  border-radius: 5px;
  margin-top: 5px;
}

.transaction-element {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.amount-container {
  text-align: right;
}

.acc-name {
  color: blue;
}

.transfer-transaction {
  font-size: 0.7em;
}

.income-transaction {
  color: green;
}

.expense-transaction {
  color: red;
}

.adjustment-transaction {
  color: #6c757d;
  font-style: italic;
}
</style>
