<script setup>
import { computed, onBeforeMount, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { DateTime } from 'luxon';

import { CREDIT_CARD_ACC_TYPE_ID } from '@/constants.js';
import { Services } from '../../services/servicesConfig';
import { processError } from '@/errors/errorHandlers';

const props = defineProps([
  'accountCreated',
  'closeNewAccForm',
  'accountDetails',
]);

const router = useRouter();

const accountType = ref(1);
const currency = ref(1);
const name = ref('');
const balance = ref(0);
const creditLimit = ref(0);
const openingDate = ref(DateTime.now().toFormat("yyyy-LL-dd'T'HH:mm"));
const comment = ref('');
const isHidden = ref(false);
const showInReports = ref(false);
const showArchiveConfirmation = ref(false);

if (props.accountDetails) {
  accountType.value = props.accountDetails.accountTypeId;
  currency.value = props.accountDetails.currencyId;
  name.value = props.accountDetails.name;
  balance.value = props.accountDetails.balance;
  creditLimit.value = props.accountDetails.creditLimit;
  openingDate.value = DateTime.fromISO(
    props.accountDetails.openingDate,
  ).toFormat("yyyy-LL-dd'T'HH:mm");
  comment.value = props.accountDetails.comment;
  isHidden.value = props.accountDetails.isHidden;
  showInReports.value = props.accountDetails.showInReports;
}

const accountTypes = reactive([]);
const currencies = reactive([]);

const formSubmitLabel = computed(() =>
  props.accountDetails ? 'Update Account' : 'Create Account',
);

onBeforeMount(async () => {
  try {
    accountTypes.splice(0);
    accountTypes.push(...(await Services.accountsService.getAccountTypes()));
    currencies.splice(0);
    currencies.push(...(await Services.currenciesService.getAllCurrencies()));
  } catch (e) {
    await processError(e, router);
  }
});

function handleFormSubmit(event) {
  event.preventDefault();
  if (props.accountDetails) {
    updateAccount();
  } else {
    createAccount();
  }
}

async function updateAccount() {
  const updatedAccount = {
    id: props.accountDetails.id,
    account_type_id: accountType.value,
    currency_id: currency.value,
    name: name.value,
    balance: balance.value,
    creditLimit: creditLimit.value,
    opening_date: openingDate.value,
    comment: comment.value,
    is_hidden: isHidden.value,
    show_in_reports: showInReports.value,
  };

  try {
    await Services.accountsService.updateAccount(updatedAccount);
    props.accountCreated();
  } catch (e) {
    await processError(e, router);
  }
}

async function createAccount() {
  const newAccount = {
    account_type_id: accountType.value,
    currency_id: currency.value,
    name: name.value,
    balance: balance.value,
    creditLimit: creditLimit.value,
    opening_date: openingDate.value,
    comment: comment.value,
    is_hidden: isHidden.value,
  };

  try {
    await Services.accountsService.createAccount(newAccount);
    props.accountCreated();
  } catch (e) {
    await processError(e, router);
  }
}

const archiveAccount = async (id) => {
  const updated = await Services.accountsService.setArchivedStatus(id, true);
  if (updated) {
    await router.push({ name: 'accounts' });
  }
};

function toggleArchiveConfirmation() {
  showArchiveConfirmation.value = true;
}
</script>

<template>
  <div class="container">
    <form
      @submit.prevent="handleFormSubmit"
      class="form">
      <div class="buttons-row">
        <a
          class="btn btn-primary"
          @click.prevent="props.closeNewAccForm"
          href="">
          Cancel
        </a>
      </div>
      <div class="form-group">
        <label for="account_type_id">Account Type:</label>
        <div class="select-wrapper">
          <select
            id="account_type_id"
            v-model="accountType"
            required>
            <option
              value=""
              disabled>
              Select an account type
            </option>
            <option
              v-for="type in accountTypes"
              :key="type.id"
              :value="type.id">
              {{ $t('accountTypes.' + type.type_name) }}
            </option>
          </select>
          <i class="arrow"></i>
        </div>
      </div>

      <div
        v-if="accountType === CREDIT_CARD_ACC_TYPE_ID"
        class="form-group">
        <label for="balance">{{ $t('message.creditLimit') }}</label>
        <input
          type="number"
          id="balance"
          v-model="creditLimit"
          step="0.01" />
      </div>

      <div class="form-group">
        <label for="currency_id">Currency:</label>
        <div class="select-wrapper">
          <select
            id="currency_id"
            v-model="currency"
            required>
            <option
              value=""
              disabled>
              Select a currency
            </option>
            <option
              v-for="currency in currencies"
              :key="currency.id"
              :value="currency.id">
              {{ currency.name }}
            </option>
          </select>
          <i class="arrow"></i>
        </div>
      </div>

      <div class="form-group">
        <label for="name">Name:</label>
        <input
          type="text"
          id="name"
          v-model="name"
          required />
      </div>
      <div class="form-group">
        <label for="balance">Balance:</label>
        <input
          type="number"
          id="balance"
          v-model="balance"
          step="0.01"
          :readonly="!!props.accountDetails"
          required />
      </div>
      <div class="form-group">
        <label for="opening_date">Opening Date:</label>
        <input
          type="datetime-local"
          id="opening_date"
          v-model="openingDate" />
      </div>
      <div class="form-group">
        <label for="comment">Comment:</label>
        <textarea
          id="comment"
          v-model="comment"></textarea>
      </div>
      <div class="form-group">
        <label for="is_hidden">Is Hidden:</label>
        <input
          type="checkbox"
          id="is_hidden"
          v-model="isHidden" />
      </div>

      <div class="form-group">
        <label for="show-in-reports">{{ $t('message.showInReports') }}</label>
        <input
          type="checkbox"
          id="show-in-reports"
          v-model="showInReports" />
      </div>

      <button type="submit">{{ formSubmitLabel }}</button>
      <button
        v-if="accountDetails"
        type="button"
        @click="toggleArchiveConfirmation"
        class="btn btn-danger">
        {{ $t('buttons.archiveAccount') }}
      </button>
    </form>

    <div
      v-if="showArchiveConfirmation"
      class="overlay"></div>

    <div
      v-if="showArchiveConfirmation"
      class="archive-confirmation">
      <p>{{ $t('buttons.archiveAccount') }}?</p>
      <div class="buttons-container">
        <button
          @click="archiveAccount(accountDetails.id)"
          class="btn btn-danger">
          {{ $t('buttons.yes') }}
        </button>
        <button
          @click="showArchiveConfirmation = false"
          class="btn btn-secondary">
          {{ $t('buttons.cancel') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.container {
  display: flex;
  justify-content: center;
  align-items: center;
}

.buttons-row {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 1rem;
  width: 100%;
}

.form {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  width: 100%;
  max-width: 500px;
}

.form-group {
  width: 100%;
}

label {
  font-weight: bold;
}

input[type='number'],
input[type='text'],
input[type='datetime-local'],
textarea {
  padding: 0.5rem;
  border-radius: 0.25rem;
  border: 1px solid #ccc;
  width: 100%;
  box-sizing: border-box;
}

input[type='number']:read-only {
  background-color: #e9ecef;
  cursor: not-allowed;
}

input[type='checkbox'] {
  margin-left: 0.5rem;
}

button[type='submit'] {
  background-color: #4caf50;
  color: #fff;
  border: none;
  border-radius: 0.25rem;
  padding: 0.5rem 1rem;
  cursor: pointer;
  transition: background-color 0.3s ease;
  width: 100%;
}

button[type='submit']:hover {
  background-color: #3e8e41;
}

/* Custom styles for select element */
.select-wrapper {
  position: relative;
  width: 100%;
}

select {
  -webkit-appearance: none;
  -moz-appearance: none;
  appearance: none;
  padding: 0.5rem;
  border-radius: 0.25rem;
  border: 1px solid #ccc;
  width: 100%;
  box-sizing: border-box;
  background-color: #fff;
  cursor: pointer;
}

select:focus {
  outline: none;
}

.arrow {
  position: absolute;
  top: 50%;
  right: 0.5rem;
  transform: translateY(-50%);
  width: 0;
  height: 0;
  border-style: solid;
  border-width: 0.25rem 0.25rem 0 0.25rem;
  border-color: #333 transparent transparent transparent;
  pointer-events: none;
}

.buttons-container {
  display: flex;
  justify-content: center;
  gap: 10px;
}

.archive-confirmation {
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

.archive-confirmation p {
  margin-bottom: 20px;
  font-size: 16px;
  color: #333;
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
