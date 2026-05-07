<script setup>
import { onBeforeMount, reactive, ref, computed, nextTick, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { DateTime } from 'luxon';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';
import { HttpError } from '@/errors/HttpError';
import { resolveBackendError } from '@/errors/resolveBackendError';

import TransactionTypeTabs from '@/components/transactions/TransactionTypeTabs.vue';
import TransactionLabel from '@/components/transactions/TransactionLabel.vue';
import TransactionAmount from '@/components/transactions/TransactionAmount.vue';
import Category from '@/components/transactions/Category.vue';
import AccountSelector from '@/components/transactions/AccountSelector.vue';
import ExchangeRate from '@/components/transactions/ExchangeRate.vue';
import TransactionDateTime from '@/components/transactions/TransactionDateTIme.vue';
import { useUserStore } from '@/stores/user';

const props = defineProps(['isEdit', 'returnUrl', 'accountId']);
const router = useRouter();
const route = useRoute();
const { t, te } = useI18n();
const userStore = useUserStore();

// Single banner shown above the form when the backend rejects a submit.
// Cleared by `clearFormError` on the next submit attempt and on any user edit.
const formError = ref('');
function clearFormError() {
  if (formError.value !== '') {
    formError.value = '';
  }
}

let accounts = reactive([]);
let currentAccount = reactive({});
let targetAccount = reactive({});
let categories = reactive([]);
const showDeleteConfirmation = ref(false);
const showSubmittedOverlay = ref(false);
const amountInput = ref(null);
const amountComponent = ref(null);

let transaction = reactive({
  label: '',
  amount: 0,
  categoryId: null,
  accountId: null,
  targetAccountId: null,
  targetAmount: null,
  isTransfer: false,
  isIncome: false,
  dateTime: null,
  notes: '',
});
const saveAsTemplate = ref(false);
let filteredCategories = reactive([]);

const itemType = ref('expense');
transaction.isTransfer = itemType.value === 'transfer';

const returnUrlName = ref('');

const isSelfTransfer = computed(() => {
  return (
    transaction.isTransfer &&
    transaction.accountId != null &&
    transaction.targetAccountId != null &&
    transaction.accountId === transaction.targetAccountId
  );
});

function changeAccount({ accountType, accountId }) {
  if (accountType === 'src') {
    transaction.accountId = accountId;
    currentAccount = accounts.find((a) => a.id === accountId) || currentAccount;
  } else if (accountType === 'target') {
    transaction.targetAccountId = accountId;
    targetAccount = accounts.find((a) => a.id === accountId) || targetAccount;
  }
}

function amountChanged({ amountType, amount }) {
  if (transaction.isTransfer) {
    if (amountType === 'src') {
      transaction.amount = amount;
    } else {
      transaction.targetAmount = amount;
    }
  } else {
    transaction.amount = amount;
  }
}

onBeforeMount(async () => {
  if (props.returnUrl === 'accountDetails') {
    returnUrlName.value = 'accountDetails';
  } else {
    returnUrlName.value = 'transactions';
  }

  try {
    categories = await Services.categoriesService.getUserCategories();
    accounts.length = 0;
    accounts.push(
      ...(await Services.accountsService.getUserAccounts({
        shouldUpdate: true,
        includeHidden: true,
      })),
    );
    userStore.transactionTemplates.splice(
      0,
      userStore.transactionTemplates.length,
      ...(await Services.transactionsService.getUserTemplates()),
    );
    if (props.accountId) {
      currentAccount = accounts.find(
        (item) => item.id === parseInt(props.accountId, 10),
      );
    } else {
      currentAccount = accounts[0];
    }
    targetAccount = accounts[0];
    transaction.accountId = currentAccount.id;
    transaction.targetAccountId = targetAccount.id;

    if (props.isEdit) {
      let details = await Services.transactionsService.getTransactionDetails(
        route.params.id,
      );
      Object.assign(transaction, details);
      itemType.value = transaction.isTransfer
        ? 'transfer'
        : transaction.isIncome
          ? 'income'
          : 'expense';

      // !!!!!!!
      // if we have an income part of transfer, we load and edit the expense part
      // it allows to simplify MUCH logic and UI
      if (transaction.isTransfer && transaction.isIncome) {
        details = await Services.transactionsService.getTransactionDetails(
          transaction.linkedTransactionId,
        );
        transaction = Object.assign(transaction, details);
      }

      currentAccount = accounts.find(
        (item) => item.id === transaction.accountId,
      );

      if (transaction.linkedTransactionId) {
        await getLinkedTransaction(transaction);
      }
    }

    filterCategories();
  } catch (e) {
    await processError(e, router);
  }
});

async function getLinkedTransaction(transaction) {
  const linkedTransaction =
    await Services.transactionsService.getTransactionDetails(
      transaction.linkedTransactionId,
    );
  transaction.targetAccountId = linkedTransaction.accountId;
  transaction.targetAmount = linkedTransaction.amount;
}

function filterCategories() {
  const isIncome = itemType.value === 'income';
  // filter categories by income/expense
  filteredCategories = categories.filter((item) => item.isIncome === isIncome);

  // if the current category is not in the filtered categories, set the first one from the filtered list
  if (!filteredCategories.some((item) => item.id === transaction.categoryId)) {
    transaction.categoryId = filteredCategories[0].id;
  }

  updateTransactionProperties(itemType.value);
}

function updateTransactionProperties(type) {
  if (type === 'transfer') {
    transaction.categoryId = null;
    transaction.isTransfer = true;
  } else {
    transaction.targetAccountId = null;
    transaction.targetAmount = null;
    transaction.isTransfer = false;
    transaction.categoryId =
      transaction.categoryId || filteredCategories.value[0].id;
    transaction.isIncome = itemType.value === 'income';
  }
}

function changeNotes($event) {
  transaction.notes = $event.target.value;
}

function changeItemType(type) {
  itemType.value = type;
  filterCategories();
}

function dateTimeChanged({ date, time }) {
  const dateTimeString = `${date}T${time}`;
  const localDateTime = DateTime.fromISO(dateTimeString, { zone: 'local' });
  transaction.dateTime = localDateTime.toUTC().toISO().replace('.000', '');
}

async function submitTransaction() {
  if (isSelfTransfer.value) {
    return;
  }
  // Clear any banner from a previous failed submit before attempting again.
  formError.value = '';
  showSubmittedOverlay.value = true; // disable submit button
  if (saveAsTemplate.value) {
    transaction.isTemplate = true;
  } else {
    transaction.isTemplate = false;
  }

  try {
    if (props.isEdit) {
      await Services.transactionsService.updateTransaction(transaction);
    } else {
      await Services.transactionsService.addTransaction(transaction);
    }

    await router.push({
      name: returnUrlName.value,
      params: {
        id: props.accountId,
      },
    });
    showSubmittedOverlay.value = false; // enable submit button
  } catch (e) {
    // 401 is the only case where we still delegate to the global handler —
    // the session is gone, the user must be sent to login. For every other
    // 4xx/5xx we keep the form mounted, dismiss the overlay so the user can
    // edit and retry, and render a single banner at the top of the form.
    if (e instanceof HttpError && e.statusCode === 401) {
      showSubmittedOverlay.value = false;
      await processError(e, router);
      return;
    }
    if (e instanceof HttpError) {
      formError.value = resolveBackendError(e.body, t, te);
    } else {
      formError.value = t('errors.transactionForm.generic');
    }
    showSubmittedOverlay.value = false;
  }
}

// Clear the banner whenever the user edits any input on the form. We watch
// the whole reactive `transaction` object (deep) plus the template checkbox
// rather than wiring up per-input handlers — equivalent UX, much smaller diff.
watch(
  () => ({ ...transaction, saveAsTemplate: saveAsTemplate.value }),
  () => clearFormError(),
  { deep: true },
);

function confirmDelete() {
  showDeleteConfirmation.value = true;
}

async function deleteTransaction() {
  try {
    await Services.transactionsService.deleteTransaction(transaction.id);
    await router.push({ name: 'transactions' });
  } catch (e) {
    await processError(e, router);
  } finally {
    showDeleteConfirmation.value = false;
  }
}

function updateLabel(value) {
  if (typeof value === 'string') {
    transaction.label = value;
  } else if (typeof value === 'object' && value !== null) {
    transaction.label = value.label;

    if (value.targetAccountId) {
      // Transfer template selected
      transaction.targetAccountId = value.targetAccountId;
      const matchingAccount = accounts.find(
        (a) => a.id === value.targetAccountId,
      );
      if (matchingAccount) {
        targetAccount = matchingAccount;
      }
      itemType.value = 'transfer';
      transaction.isTransfer = true;
      transaction.isIncome = false;
      transaction.categoryId = null;
    } else {
      // Regular (income/expense) template selected
      const selectedCategory = categories.find(
        (item) => item.id === value.categoryId,
      );
      if (selectedCategory) {
        transaction.categoryId = selectedCategory.id;
        transaction.isIncome = selectedCategory.isIncome;
        transaction.isTransfer = false;
        itemType.value = selectedCategory.isIncome ? 'income' : 'expense';
      }
      filterCategories();
    }

    nextTick(() => {
      if (amountComponent.value?.amountInput?.inputRef) {
        const inputElement = amountComponent.value.amountInput.inputRef;
        inputElement.focus();
        inputElement.select();
      }
    });
  }
}
</script>

<template>
  <main>
    <div class="container">
      <div class="row">
        <div class="col">
          <form @submit.prevent="submitTransaction">
            <div
              v-if="formError"
              class="alert alert-danger"
              role="alert"
              data-test="transaction-form-error">
              {{ formError }}
            </div>

            <TransactionTypeTabs
              @type-changed="changeItemType"
              :is-edit="props.isEdit"
              :transaction="transaction"
              :item-type="itemType" />

            <TransactionLabel
              :transaction="transaction"
              :is-transfer="itemType === 'transfer'"
              @update:label="updateLabel" />

            <div class="mb-3">
              <input
                type="checkbox"
                id="saveAsTemplate"
                v-model="saveAsTemplate"
                class="form-check-input" />
              <label
                for="saveAsTemplate"
                class="form-label ms-2">
                {{ $t('message.saveAsTemplate') }}
              </label>
            </div>

            <AccountSelector
              @account-changed="changeAccount"
              :label="$t('message.account')"
              account-type="src"
              :accountId="transaction.accountId"
              :accounts="accounts" />

            <TransactionAmount
              :label="t('message.amount')"
              ref="amountComponent"
              :amount-input="amountInput"
              type="src"
              @amount-changed="amountChanged"
              :amount="transaction.amount"
              :current-account="currentAccount" />

            <div
              v-if="itemType === 'transfer'"
              :class="{ 'self-transfer-error': isSelfTransfer }">
              <AccountSelector
                @account-changed="changeAccount"
                :label="$t('message.targetAccount')"
                account-type="target"
                :accountId="transaction.targetAccountId"
                :accounts="accounts" />
              <div
                v-if="isSelfTransfer"
                class="text-danger small mt-1">
                {{ $t('message.selfTransferError') }}
              </div>
            </div>

            <TransactionAmount
              v-if="itemType === 'transfer'"
              type="target"
              :label="t('message.amount')"
              @amount-changed="amountChanged"
              :amount="transaction.targetAmount"
              :current-account="targetAccount" />

            <ExchangeRate
              v-if="itemType === 'transfer'"
              :currency-src-code="currentAccount.currency.code"
              :src-amount="transaction.amount"
              :currency-target-code="targetAccount.currency.code"
              :target-amount="transaction.targetAmount" />

            <Category
              v-if="!transaction.isTransfer"
              :transaction="transaction"
              :categories="filteredCategories"
              @update:categoryId="transaction.categoryId = $event" />

            <TransactionDateTime
              :transaction="transaction"
              :is-edit="isEdit"
              @date-time-changed="dateTimeChanged" />

            <div class="mb-3">
              <label
                for="notes"
                class="form-label">
                {{ $t('message.notes') }}
              </label>
              <textarea
                @keyup="changeNotes"
                class="form-control"
                id="notes"
                v-model="transaction.notes"
                rows="3"></textarea>
            </div>

            <div class="flex-container">
              <button
                type="submit"
                class="btn btn-primary">
                {{ $t('buttons.submit') }}
              </button>
              <button
                @click.prevent="confirmDelete"
                class="btn btn-danger">
                {{ $t('buttons.delete') }}
              </button>
            </div>
          </form>

          <div
            v-if="showSubmittedOverlay"
            id="submitted-overlay"
            class="overlay"></div>

          <div
            v-if="showDeleteConfirmation"
            class="overlay"></div>

          <div
            v-if="showDeleteConfirmation"
            class="delete-confirmation">
            <p>Delete the transaction?</p>
            <div class="buttons-container">
              <button
                @click="deleteTransaction"
                class="btn btn-danger">
                {{ $t('buttons.yes') }}
              </button>
              <button
                @click="showDeleteConfirmation = false"
                class="btn btn-secondary">
                {{ $t('buttons.cancel') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.flex-container {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
}

.delete-confirmation {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  padding: 20px;
  background-color: white;
  border: 1px solid #ccc;
  border-radius: 5px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  z-index: 1000;
  text-align: center;
  width: auto;
}

.delete-confirmation p {
  margin-bottom: 20px;
  font-size: 16px;
  color: #333;
}

.buttons-container {
  display: flex;
  justify-content: center;
  gap: 10px;
}

.self-transfer-error :deep(.form-select) {
  border-color: #dc3545;
  box-shadow: 0 0 0 0.25rem rgba(220, 53, 69, 0.25);
}

.overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: 999;
}
</style>
