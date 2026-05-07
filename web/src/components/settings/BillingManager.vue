<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useSubscriptionStore } from '@/stores/subscription';
import { useUserStore } from '@/stores/user';
import { usePaymentStore } from '@/stores/payment';
import { SubscriptionService } from '@/services/subscriptionService';
import { UserService } from '@/services/users';
import SubscriptionUpgradeModal from '@/components/subscription/SubscriptionUpgradeModal.vue';

const { t } = useI18n();
const subscriptionStore = useSubscriptionStore();
const userStore = useUserStore();
const paymentStore = usePaymentStore();

const userService = new UserService(userStore);
const subscriptionService = new SubscriptionService(
  subscriptionStore,
  userService,
);

const isLoading = ref(false);
const isOpeningPortal = ref(false);
const showUpgradeModal = ref(false);

const subscription = computed(() => subscriptionStore.subscription);
const isPremium = computed(() => {
  const planCode = subscription.value?.plan?.plan_code;
  return planCode === 'premium_monthly' || planCode === 'premium_yearly';
});
const isFree = computed(() => subscription.value?.plan?.plan_code === 'free');
const isTrial = computed(() => subscription.value?.plan?.plan_code === 'trial');

const expirationDate = computed(() => {
  if (!subscription.value?.expiresAt) return null;
  return new Date(subscription.value.expiresAt).toLocaleDateString();
});

const subscribedDate = computed(() => {
  if (!subscription.value?.subscribedAt) return null;
  return new Date(subscription.value.subscribedAt).toLocaleDateString();
});

const hasPendingPlanChange = computed(
  () => !!subscription.value?.pendingPlanId,
);
const pendingPlanName = computed(() => subscription.value?.pendingPlanName);

onMounted(async () => {
  isLoading.value = true;
  try {
    await subscriptionService.getCurrentSubscription();
  } finally {
    isLoading.value = false;
  }
});

async function openBillingPortal() {
  isOpeningPortal.value = true;
  try {
    await paymentStore.openCustomerPortal(window.location.href);
  } catch (error) {
    console.error('[BillingManager] Failed to open billing portal:', error);
  } finally {
    isOpeningPortal.value = false;
  }
}

function openUpgradeModal() {
  showUpgradeModal.value = true;
}
</script>

<template>
  <div class="billing-manager">
    <!-- Loading state -->
    <div
      v-if="isLoading"
      class="text-center py-4">
      <div
        class="spinner-border text-primary"
        role="status">
        <span class="visually-hidden">{{ t('common.loading') }}</span>
      </div>
    </div>

    <!-- Subscription info -->
    <div
      v-else
      class="subscription-info">
      <!-- Current plan display -->
      <div class="current-plan-card mb-4">
        <h5 class="mb-3">{{ t('subscription.currentPlan') }}</h5>

        <div
          class="plan-badge"
          :class="{ premium: isPremium, free: isFree, trial: isTrial }">
          <i
            class="fa-solid"
            :class="
              isPremium ? 'fa-crown' : isTrial ? 'fa-clock' : 'fa-user'
            "></i>
          <span>{{ subscription?.plan?.name || t('subscription.free') }}</span>
        </div>

        <!-- Premium subscription details -->
        <div
          v-if="isPremium"
          class="plan-details mt-3">
          <p class="text-success mb-2">
            <i class="fa-solid fa-check-circle me-2"></i>
            {{ t('subscription.premiumActive') }}
          </p>
          <p
            v-if="subscribedDate"
            class="text-muted small mb-1">
            <i class="fa-solid fa-calendar-plus me-2"></i>
            {{ t('subscription.subscribedOn') }}: {{ subscribedDate }}
          </p>
          <p
            v-if="expirationDate"
            class="text-muted small mb-0">
            <i class="fa-solid fa-calendar-check me-2"></i>
            {{ t('subscription.expiresOn') }}: {{ expirationDate }}
          </p>

          <!-- Pending plan change notice -->
          <div
            v-if="hasPendingPlanChange"
            class="pending-change-notice mt-3">
            <p class="text-warning mb-0">
              <i class="fa-solid fa-clock-rotate-left me-2"></i>
              {{
                t('subscription.pendingPlanChange', {
                  planName: pendingPlanName,
                  date: expirationDate,
                })
              }}
            </p>
          </div>
        </div>

        <!-- Free plan details -->
        <div
          v-else-if="isFree"
          class="plan-details mt-3">
          <p class="text-muted mb-0">
            {{ t('subscription.youAreOnFreePlan') }}
          </p>
        </div>

        <!-- Trial details -->
        <div
          v-else-if="isTrial"
          class="plan-details mt-3">
          <p
            v-if="expirationDate"
            class="text-warning mb-0">
            <i class="fa-solid fa-clock me-2"></i>
            {{ t('subscription.expiresOn') }}: {{ expirationDate }}
          </p>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="billing-actions">
        <!-- Premium users: Manage Billing button -->
        <button
          v-if="isPremium"
          class="btn btn-outline-primary me-2"
          :disabled="isOpeningPortal"
          @click="openBillingPortal">
          <span v-if="isOpeningPortal">
            <span
              class="spinner-border spinner-border-sm me-2"
              role="status"></span>
            {{ t('common.loading') }}
          </span>
          <span v-else>
            <i class="fa-solid fa-credit-card me-2"></i>
            {{ t('payment.manageBilling') }}
          </span>
        </button>

        <!-- Non-premium users: Upgrade button -->
        <button
          v-if="!isPremium"
          class="btn btn-primary"
          @click="openUpgradeModal">
          <i class="fa-solid fa-arrow-up me-2"></i>
          {{ t('subscription.upgradeToPremium') }}
        </button>

        <!-- Premium users: Change plan button -->
        <button
          v-if="isPremium"
          class="btn btn-outline-secondary"
          @click="openUpgradeModal">
          <i class="fa-solid fa-exchange-alt me-2"></i>
          {{ t('subscription.changePlan') }}
        </button>
      </div>
    </div>

    <!-- Upgrade modal -->
    <SubscriptionUpgradeModal
      :show="showUpgradeModal"
      @close="showUpgradeModal = false" />
  </div>
</template>

<style scoped>
.billing-manager {
  max-width: 500px;
}

.current-plan-card {
  background: var(--bs-light, #f8f9fa);
  border-radius: 12px;
  padding: 1.5rem;
}

.plan-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 20px;
  font-weight: 600;
  font-size: 1.1rem;
}

.plan-badge.premium {
  background: linear-gradient(135deg, #ffd700, #ffb347);
  color: #5a4000;
}

.plan-badge.free {
  background: #e9ecef;
  color: #495057;
}

.plan-badge.trial {
  background: #fff3cd;
  color: #856404;
}

.plan-details {
  padding-top: 0.5rem;
}

.billing-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.pending-change-notice {
  background: #fff3cd;
  border: 1px solid #ffc107;
  border-radius: 8px;
  padding: 0.75rem 1rem;
}
</style>
