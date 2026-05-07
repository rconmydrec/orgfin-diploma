<script setup>
import { onBeforeMount, reactive, ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { DateTime } from 'luxon';

import { Services } from '../services/servicesConfig';
import { processError } from '@/errors/errorHandlers';

import TransactionsListView from './TransactionsListView.vue';
import newAccount from '../components/account/newAccount.vue';
import AccountTypeBadge from '@/components/account/AccountTypeBadge.vue';
import { CREDIT_CARD_ACC_TYPE_ID } from '@/constants.js';

const route = useRoute();
const router = useRouter();
const t = useI18n().t;

let accountDetails = reactive({});
const showConfirmation = ref(false);
const showEditAccForm = ref(false);
const showAdjustBalance = ref(false);
const newBalance = ref(0);
const adjustmentNotes = ref('');
const transactionsListKey = ref(0);

onBeforeMount(async () => {
  await loadAccountDetails();
});

async function loadAccountDetails() {
  try {
    const details = await Services.accountsService.getAccountDetails(
      route.params.id,
    );
    if (details) {
      accountDetails = Object.assign(accountDetails, details);
    }
  } catch (e) {
    await processError(e, router);
  }
}

const handleDeleteClick = () => {
  showConfirmation.value = true; // Show the confirmation popup
};

const formattedBalance = computed(() => {
  if (!accountDetails.balance) return '0.00';
  return accountDetails.balance.toLocaleString('ru-UA', {
    style: 'decimal',
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
  });
});

const handleUpdateAccount = () => {
  loadAccountDetails();
  closeEditAccForm();
};

const handleEditClick = () => {
  showEditAccForm.value = true;
};

const closeEditAccForm = () => {
  showEditAccForm.value = false;
};

const confirmDelete = async () => {
  try {
    await Services.accountsService.deleteAccount(accountDetails.id);
    await router.push({ name: 'accounts' });
  } catch (e) {
    await processError(e, router);
  } finally {
    showConfirmation.value = false; // Hide popup regardless of outcome
  }
};

const availableBalanceCC = computed(() => {
  return accountDetails.balance + accountDetails.creditLimit;
});

const balanceDifference = computed(() => {
  if (!accountDetails.balance) return newBalance.value;
  return newBalance.value - accountDetails.balance;
});

const handleAdjustBalanceClick = () => {
  newBalance.value = accountDetails.balance || 0;
  adjustmentNotes.value = '';
  showAdjustBalance.value = true;
};

const confirmAdjustBalance = async () => {
  try {
    await Services.accountsService.adjustBalance(
      accountDetails.id,
      newBalance.value,
      adjustmentNotes.value || null,
    );
    await loadAccountDetails();
    showAdjustBalance.value = false;
    // Force transactions list refresh by changing key
    transactionsListKey.value++;
  } catch (e) {
    await processError(e, router);
  }
};
</script>

<template>
  <main>
    <div class="container">
      <div class="account-row">
        <div class="account-name">
          <span>
            {{ $t('message.account') }}:
            <strong>{{ accountDetails.name }}</strong>
            <AccountTypeBadge
              v-if="accountDetails.accountTypeId"
              :type-id="accountDetails.accountTypeId"
              :show-label="true"
              class="ms-2" />
          </span>
          <a
            class="edit-acc-icon"
            href=""
            @click.prevent="handleEditClick">
            <img
              src="/images/icons/edit-icon.svg"
              alt="Edit account"
              width="24"
              height="24"
              :title="t('message.editAccount')" />
          </a>
        </div>

        <div class="account-balance">
          <span>
            {{ $t('message.balance') }}:
            <b>
              {{ formattedBalance }}
              <span
                v-if="accountDetails.accountTypeId == CREDIT_CARD_ACC_TYPE_ID">
                ({{ $n(availableBalanceCC, 'decimal') }})
              </span>
              &nbsp;{{ accountDetails?.currency?.code }}
            </b>
          </span>
          <div class="balance-actions">
            <a
              class="adjust-balance-icon"
              href=""
              @click.prevent="handleAdjustBalanceClick"
              :title="$t('message.adjustBalance')">
              <i class="fa-solid fa-plus-minus"></i>
            </a>
            <a
              class="edit-acc-icon"
              href=""
              @click.prevent="handleDeleteClick">
              <img
                src="/images/icons/delete-icon.svg"
                alt="Delete account"
                width="24"
                height="24"
                :title="t('message.deleteAccount')" />
            </a>
          </div>
        </div>
        <div class="account-open-date">
          <span>
            {{ $t('message.created') }}:
            <b>
              {{ DateTime.fromISO(accountDetails?.openingDate).toISODate() }}
            </b>
          </span>
        </div>
      </div>
    </div>

    <div
      v-if="showEditAccForm"
      class="row">
      <div class="col">
        <newAccount
          :account-created="handleUpdateAccount"
          :close-new-acc-form="closeEditAccForm"
          :account-details="accountDetails" />
      </div>
    </div>

    <TransactionsListView
      :key="transactionsListKey"
      :account-id="route.params.id"
      :is-account-details="true" />

    <div
      v-if="showConfirmation"
      class="confirmation-popup-overlay">
      <div class="confirmation-popup">
        <p>Are you sure want to delete this account?</p>
        <a
          @click="confirmDelete"
          class="btn btn-primary">
          Yes
        </a>
        <a
          @click="showConfirmation = false"
          class="btn btn-secondary">
          No
        </a>
      </div>
    </div>

    <div
      v-if="showAdjustBalance"
      class="confirmation-popup-overlay">
      <div class="confirmation-popup adjust-balance-popup">
        <h4>{{ $t('message.adjustBalance') }}</h4>
        <div class="form-group">
          <label>{{ $t('message.currentBalance') }}:</label>
          <span class="current-balance">
            {{ formattedBalance }} {{ accountDetails?.currency?.code }}
          </span>
        </div>
        <div class="form-group">
          <label for="newBalance">{{ $t('message.newBalance') }}:</label>
          <input
            type="number"
            id="newBalance"
            v-model.number="newBalance"
            step="0.01"
            class="form-control" />
        </div>
        <div
          class="form-group difference"
          :class="{
            positive: balanceDifference > 0,
            negative: balanceDifference < 0,
          }">
          <label>{{ $t('message.difference') }}:</label>
          <span>
            {{ balanceDifference >= 0 ? '+' : ''
            }}{{ balanceDifference.toFixed(2) }}
            {{ accountDetails?.currency?.code }}
          </span>
        </div>
        <div class="form-group">
          <label for="adjustmentNotes">{{ $t('message.notes') }}:</label>
          <input
            type="text"
            id="adjustmentNotes"
            v-model="adjustmentNotes"
            class="form-control"
            :placeholder="$t('message.optional')" />
        </div>
        <div class="popup-buttons">
          <a
            @click="confirmAdjustBalance"
            class="btn btn-primary"
            :disabled="balanceDifference === 0">
            {{ $t('message.confirm') }}
          </a>
          <a
            @click="showAdjustBalance = false"
            class="btn btn-secondary">
            {{ $t('message.cancel') }}
          </a>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.container {
  margin: 0 auto;
}

.account-row {
  background-color: #e2e0e0;
  padding: 5px 10px;
  border-radius: 5px;
  margin: 0.5rem 0 0.5rem 0.5rem;
}

.account-name,
.account-balance {
  font-size: 1em;
  color: #333;
  display: flex;
  justify-content: space-between;
  width: 100%;
}

.delete-acc-icon {
  display: inline-block;
  text-decoration: none;
}

.confirmation-popup-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.confirmation-popup {
  background-color: #fff;
  padding: 20px;
  border-radius: 10px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  z-index: 1001;
  width: 90%;
  max-width: 400px;
}

.btn-primary {
  margin-right: 8px;
}

.balance-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.adjust-balance-icon {
  color: #555;
  font-size: 1.1em;
  text-decoration: none;
}

.adjust-balance-icon:hover {
  color: #007bff;
}

.adjust-balance-popup {
  text-align: left;
}

.adjust-balance-popup h4 {
  margin-bottom: 15px;
  text-align: center;
}

.adjust-balance-popup .form-group {
  margin-bottom: 12px;
}

.adjust-balance-popup label {
  display: block;
  margin-bottom: 4px;
  font-weight: 500;
}

.adjust-balance-popup .form-control {
  width: 100%;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
}

.adjust-balance-popup .current-balance {
  font-weight: bold;
}

.adjust-balance-popup .difference {
  padding: 8px;
  border-radius: 4px;
  background-color: #f5f5f5;
}

.adjust-balance-popup .difference.positive span {
  color: #28a745;
  font-weight: bold;
}

.adjust-balance-popup .difference.negative span {
  color: #dc3545;
  font-weight: bold;
}

.popup-buttons {
  margin-top: 15px;
  text-align: center;
}
</style>
