<script setup>
import { onBeforeMount, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';
import newAccount from '@/components/account/newAccount.vue';
import ArchivedAccountsList from '@/components/account/ArchivedAccountsList.vue';
import MainAccountsList from '@/components/account/MainAccountsList.vue';
import { DateTime } from 'luxon';
const router = useRouter();

let accounts = reactive([]);
let archivedAccounts = reactive([]);

const showNewAccForm = ref(false);
const showHiddenAccounts = ref(false);
const showArchivedAccounts = ref(false);
const showMainAccountsList = ref(true);

const totalBalance = ref(0);
const baseCurrencyCode = ref('');
const today = DateTime.now().toISODate();

onBeforeMount(async () => {
  try {
    await reReadAllAccounts(true);
  } catch (e) {
    await processError(e, router);
  }
});

async function getArchivedAccounts(shouldUpdate = true) {
  try {
    const params = {
      shouldUpdate: shouldUpdate,
      archivedOnly: true,
    };
    archivedAccounts.length = 0;
    const tmpAccounts = await Services.accountsService.getUserAccounts(params);
    if (tmpAccounts) {
      archivedAccounts.push(...tmpAccounts);
    }

    if (archivedAccounts.length === 0) {
      showArchivedAccounts.value = false;
      showMainAccountsList.value = true;
    }
  } catch (e) {
    await processError(e, router);
  }
}

async function reReadAllAccounts(shouldUpdate = false) {
  accounts.length = 0;
  try {
    const params = {
      includeHidden: showHiddenAccounts.value,
      shouldUpdate: shouldUpdate,
      includeArchived: false,
      archivedOnly: false,
    };
    const tmpAccounts = await Services.accountsService.getUserAccounts(params);
    if (tmpAccounts) {
      accounts.push(...tmpAccounts);
    }

    await getArchivedAccounts();

    totalBalance.value = accounts.reduce(
      (acc, item) => acc + item.balanceInBaseCurrency,
      0,
    );

    if (localStorage.getItem('baseCurrency')) {
      baseCurrencyCode.value = localStorage.getItem('baseCurrency');
    } else {
      baseCurrencyCode.value = await Services.settingsService.getBaseCurrency();
    }
  } catch (e) {
    await processError(e, router);
  }
}

async function updateAccountsList(event) {
  if (event) {
    event.preventDefault();
  }
  await reReadAllAccounts(true);
}

async function accountCreated() {
  await updateAccountsList();
  closeNewAccForm();
}

function closeNewAccForm() {
  showNewAccForm.value = false;
}

function toggleHiddenAccounts(event) {
  showHiddenAccounts.value = event.target.checked;
  reReadAllAccounts(true);
}

function toggleArchivedAccounts(event) {
  showArchivedAccounts.value = event.target.checked;
  showMainAccountsList.value = !event.target.checked;
}
</script>

<template>
  <main>
    <div class="container">
      <div class="row">
        <div class="col-auto me-auto">
          <label
            v-if="!showArchivedAccounts"
            class="btn btn-secondary me-2">
            <input
              type="checkbox"
              @change="toggleHiddenAccounts" />
            {{ $t('message.showHiddenAccounts') }}
          </label>
          <label
            v-if="archivedAccounts.length > 0"
            class="btn btn-secondary">
            <input
              type="checkbox"
              @change="toggleArchivedAccounts" />
            {{ $t('buttons.showArchivedAccounts') }}
          </label>
        </div>
        <div class="col-auto sub-menu">
          <a
            href="javascript:void(0);"
            class="btn btn-secondary"
            @click.prevent="showNewAccForm = !showNewAccForm"
            :title="$t('message.newAccount')">
            <span class="icon-text">✚</span>
          </a>
        </div>
      </div>

      <div
        v-if="showNewAccForm"
        class="row">
        <div class="col">
          <newAccount
            :account-created="accountCreated"
            :close-new-acc-form="closeNewAccForm" />
        </div>
      </div>

      <div v-if="showArchivedAccounts">
        <archivedAccountsList
          :archived-accounts="archivedAccounts"
          :re-read-all-accounts="reReadAllAccounts" />
      </div>

      <div v-if="showMainAccountsList">
        <mainAccountsList
          :accounts="accounts"
          :total-balance="totalBalance"
          :base-currency-code="baseCurrencyCode" />
      </div>
    </div>
  </main>
</template>

<style scoped lang="scss">
@use '@/assets/main.scss' as *;

.container > .row:first-child {
  margin-top: 12px;
}

.icon-text {
  font-size: 0.95rem;
  line-height: 1;
  display: inline-block;
  vertical-align: middle;
}

.btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.875rem;
}

label.btn {
  margin-bottom: 0;
}
</style>
