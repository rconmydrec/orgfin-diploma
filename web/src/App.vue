<script setup>
import {
  computed,
  onBeforeMount,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue';
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';

import { useUserStore } from '@/stores/user';
import { useSubscriptionStore } from '@/stores/subscription';
import { useAccountStore } from '@/stores/account';
import { UserService } from '@/services/users';
import { SubscriptionService } from '@/services/subscriptionService';
import { AccountService } from '@/services/accounts';
import { BudgetsService } from '@/services/budgets';
import { SettingsService } from '@/services/settings';
import SubscriptionBadge from '@/components/subscription/SubscriptionBadge.vue';
import SubscriptionExpiredModal from '@/components/subscription/SubscriptionExpiredModal.vue';
import SubscriptionUpgradeModal from '@/components/subscription/SubscriptionUpgradeModal.vue';
import LanguageSwitcher from '@/components/LanguageSwitcher.vue';

const router = useRouter();
const route = useRoute();
const { locale, t } = useI18n();

const appEnv = import.meta.env.VITE_APP_ENV || '';

const userStore = useUserStore();
const accountStore = useAccountStore();
const userService = new UserService(userStore);
const accountService = new AccountService(accountStore, userService);
const budgetsService = new BudgetsService(userService);
const settingsService = new SettingsService(userService);
const subscriptionStore = useSubscriptionStore();
const subscriptionService = new SubscriptionService(
  subscriptionStore,
  userService,
);

const showExpiredModal = ref(false);
const showUpgradeModal = ref(false);
const accounts = ref([]);
const budgets = ref([]);

const publicPages = ['login', 'register', 'home'];
const hideHeader = computed(() => publicPages.includes(route.name));

onBeforeMount(() => {
  try {
    const usr = JSON.parse(localStorage.getItem('user'));
    const ok = JSON.parse(localStorage.getItem('isLoggedIn'));
    const tok = localStorage.getItem('accessToken');
    if (ok) {
      locale.value = usr.settings.language;
      userService.setUser(usr, ok, tok);
    }
  } catch {
    userService.logOutUser();
  }
});

// Function to check subscription and show modal if needed
async function checkSubscriptionStatus() {
  try {
    const subscription = await subscriptionService.getCurrentSubscription();
    const status = await subscriptionService.checkSubscriptionStatus();

    if (status?.requiresDowngrade) {
      // Load accounts and budgets for the modal
      try {
        const loadedAccounts = await accountService.getUserAccounts({
          includeHidden: false,
          includeArchived: false,
          archivedOnly: false,
          shouldUpdate: true,
        });
        accounts.value = loadedAccounts || [];

        const loadedBudgets = await budgetsService.getUserBudgets('active');
        budgets.value = loadedBudgets || [];

        showExpiredModal.value = true;
      } catch (error) {
        console.error('Failed to load accounts/budgets:', error);
      }
    }
  } catch (error) {
    console.error('Failed to load subscription:', error);
  }
}

// Watch for login state changes
watch(
  () => userStore.isLoggedIn,
  async (newValue, oldValue) => {
    if (newValue && !oldValue) {
      // User just logged in
      await checkSubscriptionStatus();
    }
  },
);

onMounted(async () => {
  if (userStore.isLoggedIn) {
    userStore.startTimer();
    await checkSubscriptionStatus();
  }
});
onBeforeUnmount(() => userStore.stopTimer());

const goToSettings = () => router.push({ name: 'settings' });

async function handleLanguageChange(langCode) {
  try {
    userStore.settings.language = langCode;
    await settingsService.saveUserSettings();
  } catch (err) {
    console.error('Failed to save language:', err);
  }
}

function handleBadgeClick() {
  showUpgradeModal.value = true;
}

async function handleUpgradeCompleted() {
  showUpgradeModal.value = false;
  await checkSubscriptionStatus();
}

async function handleDowngrade(data) {
  try {
    await subscriptionService.downgradeToFree(data.accountIds, data.budgetId);
    showExpiredModal.value = false;
    // Reload the page to refresh data
    window.location.reload();
  } catch (error) {
    console.error('Downgrade failed:', error);
    throw error;
  }
}
</script>

<template>
  <header
    v-if="!hideHeader"
    class="navbar navbar-two-row">
    <div class="container nav-inner">
      <!-- Row 1: Logo on left, Lang/Plan/Timer/Settings on right -->
      <div class="nav-row nav-row-top">
        <div class="nav-left">
          <RouterLink
            to="/"
            class="logo"
            :title="t('menu.home')">
            <img
              src="/images/logo.png"
              alt="Budget Tracker"
              class="logo-img" />
          </RouterLink>
          <span
            v-if="appEnv"
            class="env-badge">
            {{ appEnv }}
          </span>
        </div>

        <div
          class="nav-right"
          v-if="userStore.isLoggedIn">
          <LanguageSwitcher @change="handleLanguageChange" />
          <SubscriptionBadge @click="handleBadgeClick" />
          <span class="time">{{ userStore.timeLeft }}</span>
          <button
            class="btn btn-icon btn-outline-primary"
            @click="goToSettings"
            :title="t('message.settings')">
            <i class="fa-solid fa-gear"></i>
          </button>
        </div>
      </div>

      <!-- Row 2: Navigation menu -->
      <nav class="nav-row nav-row-menu">
        <template v-if="!userStore.isLoggedIn">
          <RouterLink
            :to="{ name: 'home' }"
            class="btn btn-icon"
            :title="t('menu.home')">
            <i class="fa-solid fa-house"></i>
          </RouterLink>
          <RouterLink
            :to="{ name: 'login' }"
            class="btn btn-icon"
            :title="t('menu.login')">
            <i class="fa-solid fa-right-to-bracket"></i>
          </RouterLink>
          <RouterLink
            :to="{ name: 'register' }"
            class="btn btn-icon"
            :title="t('menu.register')">
            <i class="fa-solid fa-user-plus"></i>
          </RouterLink>
        </template>

        <template v-else>
          <RouterLink
            :to="{ name: 'accounts' }"
            class="btn btn-icon"
            :title="t('menu.accounts')">
            <i class="fa-solid fa-wallet"></i>
          </RouterLink>
          <RouterLink
            :to="{ name: 'transactions' }"
            class="btn btn-icon"
            :title="t('menu.transactions')">
            <i class="fa-solid fa-right-left"></i>
          </RouterLink>
          <RouterLink
            :to="{ name: 'reports' }"
            class="btn btn-icon"
            :title="t('menu.reports')">
            <i class="fa-solid fa-chart-pie"></i>
          </RouterLink>
          <RouterLink
            :to="{ name: 'budgets' }"
            class="btn btn-icon"
            :title="t('menu.budgets')">
            <i class="fa-solid fa-list-check"></i>
          </RouterLink>
          <RouterLink
            :to="{ name: 'planning' }"
            class="btn btn-icon"
            :title="t('menu.planning')">
            <i class="fa-solid fa-calendar-days"></i>
          </RouterLink>
          <RouterLink
            :to="{ name: 'logout' }"
            class="btn btn-icon"
            :title="t('menu.logout')">
            <i class="fa-solid fa-right-from-bracket"></i>
          </RouterLink>
        </template>
      </nav>
    </div>
  </header>

  <SubscriptionExpiredModal
    :show="showExpiredModal"
    :accounts="accounts"
    :budgets="budgets"
    @downgrade="handleDowngrade" />

  <SubscriptionUpgradeModal
    :show="showUpgradeModal"
    @close="showUpgradeModal = false"
    @upgraded="handleUpgradeCompleted" />

  <RouterView />
</template>

<style src="@/assets/style.css"></style>
