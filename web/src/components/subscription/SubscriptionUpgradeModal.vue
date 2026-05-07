<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useSubscriptionStore } from '@/stores/subscription';
import { usePaymentStore } from '@/stores/payment';
import { SubscriptionService } from '@/services/subscriptionService';
import { UserService } from '@/services/users';
import { useUserStore } from '@/stores/user';
import { useAccountStore } from '@/stores/account';
import { AccountService } from '@/services/accounts';
import { BudgetsService } from '@/services/budgets';
import ConfirmDialog from '@/components/utils/ConfirmDialog.vue';

const { t } = useI18n();
const subscriptionStore = useSubscriptionStore();
const paymentStore = usePaymentStore();
const userStore = useUserStore();
const accountStore = useAccountStore();
const userService = new UserService(userStore);
const subscriptionService = new SubscriptionService(
  subscriptionStore,
  userService,
);
const accountService = new AccountService(accountStore, userService);
const budgetsService = new BudgetsService(userService);

const props = defineProps({
  show: {
    type: Boolean,
    required: true,
  },
});

const emit = defineEmits(['close', 'upgraded']);

const availablePlans = ref([]);
const selectedPlanId = ref(null);
const isLoading = ref(false);
const isRedirectingToStripe = ref(false);
const error = ref('');
const planChangeWarningMessage = ref(null);
const originalPlanId = ref(null); // Capture plan ID when modal opens

// Downgrade to free
const showDowngradeForm = ref(false);
const accounts = ref([]);
const budgets = ref([]);
const selectedAccountIds = ref([]);
const selectedBudgetId = ref(null);
const isLoadingDowngrade = ref(false);
const showConfirmDialog = ref(false);
const confirmDialogMessage = ref('');
const showPlanChangeWarningDialog = ref(false);
const isCancellingSchedule = ref(false);
const planChangeCompleted = ref(false); // Flag to prevent warning after plan change
const showCancelConfirmDialog = ref(false); // Confirmation before cancelling scheduled change

const subscription = computed(() => subscriptionStore.subscription);
const isFree = computed(() => subscriptionStore.isFree);
const isPremium = computed(() => subscriptionStore.isPremium);
const expiresAt = computed(() => subscription.value?.expiresAt);
const subscribedAt = computed(() => subscription.value?.subscribedAt);
const hasStripeSubscription = computed(
  () => subscription.value?.hasStripeSubscription,
);

// Get limits from subscription store (loaded via API)
const accountLimit = computed(() => subscriptionStore.accountsLimit || 2);
const budgetLimit = computed(() => subscriptionStore.budgetsLimit || 1);

// Filtered active accounts and budgets
const activeAccounts = computed(() =>
  accounts.value.filter((acc) => !acc.isDeleted && !acc.isArchived),
);

const activeBudgets = computed(() =>
  budgets.value.filter((budget) => !budget.isDeleted),
);

// Check if user needs to select entities (exceeds Free plan limits)
const needsAccountSelection = computed(
  () => activeAccounts.value.length > accountLimit.value,
);

const needsBudgetSelection = computed(
  () => activeBudgets.value.length > budgetLimit.value,
);

// Check if any selection is needed at all
const needsAnySelection = computed(
  () => needsAccountSelection.value || needsBudgetSelection.value,
);

// Pending plan change (scheduled downgrade)
const pendingPlanId = computed(() => subscription.value?.pendingPlanId);
const pendingPlanName = computed(() => subscription.value?.pendingPlanName);
const hasPendingPlanChange = computed(() => !!pendingPlanId.value);

// Calculate remaining days and credit from current subscription
const remainingDays = computed(() => {
  if (!isPremium.value || !expiresAt.value) return 0;
  const now = new Date();
  const expires = new Date(expiresAt.value);
  const diff = expires - now;
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)));
});

const currentPlanId = computed(() => subscription.value?.planId);

const currentPlanName = computed(() => {
  if (!currentPlanId.value) return null;
  const plan = availablePlans.value.find((p) => p.id === currentPlanId.value);
  return plan?.name || null;
});

// Calculate credit from remaining days
function calculateCredit(currentPlan, remainingDays) {
  if (!currentPlan || remainingDays <= 0 || !currentPlan.billingPeriod)
    return 0;
  const dailyRate = currentPlan.price / currentPlan.billingPeriod.durationDays;
  return dailyRate * remainingDays;
}

// Calculate adjusted price for upgrade
function getAdjustedPrice(plan) {
  if (!isPremium.value || !currentPlanId.value) {
    return plan.price;
  }

  const currentPlan = availablePlans.value.find(
    (p) => p.id === currentPlanId.value,
  );
  if (!currentPlan || currentPlan.id === plan.id) {
    return plan.price;
  }

  const credit = calculateCredit(currentPlan, remainingDays.value);
  return Math.max(0, plan.price - credit);
}

// Check if plan should be disabled
function isPlanDisabled(plan) {
  // If there's a pending plan change, disable ALL plans
  if (hasPendingPlanChange.value) {
    return true;
  }
  // If user has this plan active, disable it
  return isPremium.value && currentPlanId.value === plan.id;
}

// Check if should show warning about plan change
function shouldShowPlanChangeWarning(plan) {
  // Only show warning for "plan change" (not upgrade):
  // - User has active premium with remaining paid days
  // - Selecting a different plan
  // - New plan duration < remaining days (plan will take effect AFTER current period ends)

  // Use originalPlanId (captured when modal opened) instead of currentPlanId
  // because the subscription might be updated optimistically in the store
  if (!isPremium.value || remainingDays.value <= 0 || !originalPlanId.value) {
    return false;
  }

  // Don't show warning if selecting the same plan as the original
  if (plan.id === originalPlanId.value) {
    return false;
  }

  // Check if this is a "plan change" (not upgrade)
  // Plan change: new plan duration < remaining days
  // Upgrade: new plan duration >= remaining days (applies credit, changes immediately)
  if (!plan.billingPeriod) {
    return false;
  }

  const newPlanDays = plan.billingPeriod.durationDays;
  const isUpgrade = newPlanDays >= remainingDays.value;

  // Show warning only for plan change (not upgrade)
  return !isUpgrade;
}

// Calculate warning message when plan selection changes
function updatePlanChangeWarning() {
  // Don't update if modal is not shown or plan change already completed
  if (!props.show || planChangeCompleted.value) {
    return;
  }

  // If no plan is selected, clear the warning
  if (!selectedPlanId.value) {
    planChangeWarningMessage.value = null;
    showPlanChangeWarningDialog.value = false;
    return;
  }

  // If data is temporarily missing but we already have a warning, keep it
  // This handles the case where expiresAt becomes undefined temporarily during store updates
  if (!expiresAt.value) {
    return;
  }

  const selectedPlan = availablePlans.value.find(
    (p) => p.id === selectedPlanId.value,
  );
  if (!selectedPlan) {
    return;
  }

  const shouldWarn = shouldShowPlanChangeWarning(selectedPlan);

  if (shouldWarn) {
    const message = t('subscription.planChangeWarning', {
      date: formatDate(expiresAt.value),
    });
    planChangeWarningMessage.value = message;
    // Show warning dialog only if main modal is shown
    showPlanChangeWarningDialog.value = true;
  } else {
    planChangeWarningMessage.value = null;
    showPlanChangeWarningDialog.value = false;
  }
}

watch(
  () => props.show,
  async (newValue) => {
    if (newValue) {
      // Capture the current plan ID when modal opens - use this for comparison
      // instead of reactive currentPlanId which might change during selection
      originalPlanId.value = currentPlanId.value;

      // Load plans when modal opens
      await loadPlans();
    } else {
      // Close all child dialogs and reset state when main modal closes
      showPlanChangeWarningDialog.value = false;
      showConfirmDialog.value = false;
      selectedPlanId.value = null;
      planChangeWarningMessage.value = null;
      showDowngradeForm.value = false;
      planChangeCompleted.value = false;
    }
  },
);

// Update warning when plan selection or subscription data changes
watch(selectedPlanId, () => {
  updatePlanChangeWarning();
});

watch(expiresAt, () => {
  updatePlanChangeWarning();
});

watch(subscription, () => {
  updatePlanChangeWarning();
});

async function loadPlans() {
  try {
    isLoading.value = true;
    error.value = '';
    const plans = await subscriptionService.getAvailablePlans();
    // Filter out FREE and TRIAL plans - they should never be selectable
    // FREE: only via "Downgrade to Free" button with account/budget selection
    // TRIAL: auto-assigned at registration only
    availablePlans.value = (plans || []).filter(
      (plan) => plan.planType !== 'trial' && plan.planType !== 'free',
    );
  } catch (err) {
    console.error('[SubscriptionUpgradeModal] Failed to load plans:', err);
    error.value = t('subscription.failedToLoadPlans');
  } finally {
    isLoading.value = false;
  }
}

async function handleUpgrade() {
  if (!selectedPlanId.value) {
    error.value = t('subscription.selectPlan');
    return;
  }

  try {
    isLoading.value = true;
    error.value = '';

    // If user already has Stripe subscription, use change-plan endpoint
    if (hasStripeSubscription.value) {
      // Close warning dialog and prevent it from reopening
      showPlanChangeWarningDialog.value = false;
      planChangeCompleted.value = true;

      const result = await paymentStore.changePlan(selectedPlanId.value);

      // Refresh subscription data first
      await subscriptionService.getCurrentSubscription();

      // Show localized result message
      if (result.isUpgrade) {
        confirmDialogMessage.value = t('subscription.upgradeSuccess');
      } else {
        confirmDialogMessage.value = t('subscription.planChangeScheduled', {
          planName: result.newPlanName,
          date: formatDate(result.effectiveDate),
        });
      }
      showConfirmDialog.value = true;

      isLoading.value = false;
      return;
    }

    // For new subscriptions, redirect to Stripe Checkout
    isRedirectingToStripe.value = true;
    await paymentStore.initiateCheckout(selectedPlanId.value);

    // Note: If successful, user will be redirected to Stripe
    // This code only runs if there's an error before redirect
  } catch (err) {
    console.error('Upgrade failed:', err);
    error.value = err.message || t('subscription.upgradeError');
    isLoading.value = false;
    isRedirectingToStripe.value = false;
  }
}

function formatDate(dateString) {
  if (!dateString) return '';
  const date = new Date(dateString);
  return date.toLocaleDateString();
}

function formatPrice(price, currencyCode) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currencyCode || 'USD',
  }).format(price);
}

async function toggleDowngradeForm() {
  showDowngradeForm.value = !showDowngradeForm.value;
  if (showDowngradeForm.value) {
    await loadAccountsAndBudgets();
  }
}

async function loadAccountsAndBudgets() {
  try {
    isLoadingDowngrade.value = true;
    const [accountsData, budgetsData] = await Promise.all([
      accountService.getUserAccounts({
        includeHidden: false,
        shouldUpdate: true,
        includeArchived: false,
      }),
      budgetsService.getUserBudgets('all'),
    ]);
    accounts.value = accountsData || [];
    budgets.value = budgetsData || [];
  } catch (err) {
    console.error(
      '[SubscriptionUpgradeModal] Failed to load accounts/budgets:',
      err,
    );
    error.value = t('subscription.downgradeError');
  } finally {
    isLoadingDowngrade.value = false;
  }
}

function toggleAccountSelection(accountId) {
  const index = selectedAccountIds.value.indexOf(accountId);
  if (index > -1) {
    selectedAccountIds.value.splice(index, 1);
  } else {
    if (selectedAccountIds.value.length < accountLimit.value) {
      selectedAccountIds.value.push(accountId);
    }
  }
}

// Check if downgrade can be submitted
const canSubmitDowngrade = computed(() => {
  // If user has <= limit accounts, no selection needed for accounts
  const accountsOk =
    !needsAccountSelection.value ||
    selectedAccountIds.value.length === accountLimit.value;

  // If user has <= limit budgets, no selection needed for budgets
  const budgetsOk =
    !needsBudgetSelection.value || selectedBudgetId.value !== null;

  return accountsOk && budgetsOk;
});

async function handleDowngrade() {
  if (!canSubmitDowngrade.value) {
    error.value = t('subscription.selectRequiredEntities');
    return;
  }

  try {
    isLoadingDowngrade.value = true;
    error.value = '';

    const result = await subscriptionService.downgradeToFree(
      // Send empty array if no account selection needed (user already within limits)
      needsAccountSelection.value ? selectedAccountIds.value : [],
      // Send null if no budget selection needed (user already within limits)
      needsBudgetSelection.value ? selectedBudgetId.value : null,
    );

    if (result.pending_downgrade) {
      // Scheduled downgrade
      confirmDialogMessage.value = t('subscription.downgradeScheduled', {
        date: formatDate(result.expires_at),
      });
    } else {
      // Immediate downgrade
      confirmDialogMessage.value = t('subscription.downgradeSuccess');
    }

    showConfirmDialog.value = true;
  } catch (err) {
    console.error('Downgrade failed:', err);
    error.value = err.message || t('subscription.downgradeError');
  } finally {
    isLoadingDowngrade.value = false;
  }
}

function handleConfirmDialogClose() {
  showConfirmDialog.value = false;
  emit('upgraded'); // Reload subscription data
  emit('close');
}

function promptCancelScheduledChange() {
  showCancelConfirmDialog.value = true;
}

async function cancelScheduledChange() {
  showCancelConfirmDialog.value = false;
  try {
    isCancellingSchedule.value = true;
    error.value = '';

    await paymentStore.cancelScheduledChange();

    // Refresh subscription data
    await subscriptionService.getCurrentSubscription();

    // Show success message
    confirmDialogMessage.value = t('subscription.scheduledChangeCancelled');
    showConfirmDialog.value = true;
  } catch (err) {
    console.error('Cancel scheduled change failed:', err);
    error.value = err.message || t('subscription.cancelScheduledChangeError');
  } finally {
    isCancellingSchedule.value = false;
  }
}
</script>

<template>
  <div
    v-if="show"
    class="modal-backdrop"
    @click.self="emit('close')">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content">
        <!-- Header -->
        <div class="modal-header">
          <h5 class="modal-title">
            <i
              class="fa-solid fa-crown me-2"
              style="color: #ffd700"></i>
            {{ t('subscription.changePlan') }}
          </h5>
          <button
            type="button"
            class="btn-close btn-close-white"
            @click="emit('close')"></button>
        </div>

        <!-- Body -->
        <div class="modal-body">
          <!-- Current Subscription Info -->
          <div class="subscription-info mb-4">
            <h6 class="text-center mb-3">
              {{ t('subscription.currentPlan') }}:
              <strong>
                {{
                  currentPlanId && availablePlans.length > 0
                    ? availablePlans.find((p) => p.id === currentPlanId)
                        ?.translationKey
                      ? t(
                          availablePlans.find((p) => p.id === currentPlanId)
                            .translationKey,
                        )
                      : availablePlans.find((p) => p.id === currentPlanId)?.name
                    : t(`subscription.${subscription?.planType || 'free'}`)
                }}
              </strong>
            </h6>

            <div
              v-if="subscribedAt || expiresAt"
              class="text-center small text-muted">
              <div v-if="subscribedAt">
                {{ t('subscription.subscribedOn') }}:
                {{ formatDate(subscribedAt) }}
              </div>
              <div v-if="expiresAt">
                {{ t('subscription.expiresOn') }}: {{ formatDate(expiresAt) }}
              </div>
            </div>
          </div>

          <!-- Pending Plan Change Notice -->
          <div
            v-if="hasPendingPlanChange"
            class="pending-change-notice mb-4">
            <div class="alert alert-warning mb-0">
              <i class="fa-solid fa-clock-rotate-left me-2"></i>
              <strong>{{ t('subscription.scheduledChange') }}</strong>
              <p class="mb-2 mt-2">
                {{
                  t('subscription.pendingPlanChange', {
                    planName: pendingPlanName,
                    date: formatDate(expiresAt),
                  })
                }}
              </p>
              <button
                type="button"
                class="btn btn-sm btn-outline-warning"
                @click.stop="promptCancelScheduledChange"
                :disabled="isCancellingSchedule">
                <span
                  v-if="isCancellingSchedule"
                  class="spinner-border spinner-border-sm me-1"></span>
                <i
                  v-else
                  class="fa-solid fa-times me-1"></i>
                {{ t('subscription.cancelScheduledChange') }}
              </button>
            </div>
          </div>

          <!-- Available Plans -->
          <div class="plans-section">
            <h6 class="text-center mb-3">{{ t('subscription.selectPlan') }}</h6>

            <div
              v-if="isLoading"
              class="text-center">
              <div
                class="spinner-border text-primary"
                role="status">
                <span class="visually-hidden">{{ t('common.loading') }}</span>
              </div>
            </div>

            <div
              v-else
              class="plans-grid">
              <div
                v-for="plan in availablePlans"
                :key="plan.id"
                class="plan-card"
                :class="{
                  selected: selectedPlanId === plan.id,
                  featured: plan.isFeatured,
                  disabled: isPlanDisabled(plan),
                }"
                @click="!isPlanDisabled(plan) && (selectedPlanId = plan.id)">
                <div
                  v-if="plan.isFeatured"
                  class="featured-badge">
                  <i class="fa-solid fa-star me-1"></i>
                  {{ t('subscription.mostPopular') }}
                </div>

                <h5 class="plan-name">
                  {{ plan.translationKey ? t(plan.translationKey) : plan.name }}
                </h5>

                <div class="plan-price">
                  <div
                    v-if="currentPlanId === plan.id"
                    class="current-plan-badge">
                    {{ t('subscription.currentPlan') }}
                  </div>
                  <div
                    v-else-if="pendingPlanId === plan.id"
                    class="scheduled-plan-badge">
                    {{ t('subscription.scheduledPlan') }}
                  </div>
                  <template v-else>
                    <div
                      v-if="getAdjustedPrice(plan) < plan.price"
                      class="price-with-discount">
                      <span class="original-price">
                        {{ formatPrice(plan.price, plan.currencyCode) }}
                      </span>
                      <span class="price-amount discounted">
                        {{
                          formatPrice(getAdjustedPrice(plan), plan.currencyCode)
                        }}
                      </span>
                      <span
                        v-if="plan.billingPeriod"
                        class="price-period">
                        / {{ plan.billingPeriod.name }}
                      </span>
                      <div class="credit-info">
                        {{
                          t('subscription.creditApplied', {
                            days: remainingDays,
                          })
                        }}
                      </div>
                    </div>
                    <div v-else>
                      <span class="price-amount">
                        {{ formatPrice(plan.price, plan.currencyCode) }}
                      </span>
                      <span
                        v-if="plan.billingPeriod"
                        class="price-period">
                        / {{ plan.billingPeriod.name }}
                      </span>
                    </div>
                  </template>
                </div>

                <p class="plan-description">{{ plan.description }}</p>

                <div class="plan-features">
                  <div class="feature">
                    <i class="fa-solid fa-check text-success me-2"></i>
                    {{ t('subscription.unlimitedAccounts') }}
                  </div>
                  <div class="feature">
                    <i class="fa-solid fa-check text-success me-2"></i>
                    {{ t('subscription.unlimitedBudgets') }}
                  </div>
                  <div class="feature">
                    <i class="fa-solid fa-check text-success me-2"></i>
                    {{ t('subscription.unlimitedPlanning') }}
                  </div>
                </div>

                <div
                  class="form-check mt-3"
                  v-if="!isPlanDisabled(plan)">
                  <input
                    class="form-check-input"
                    type="radio"
                    name="planSelection"
                    :id="`plan-${plan.id}`"
                    :value="plan.id"
                    v-model="selectedPlanId" />
                  <label
                    class="form-check-label"
                    :for="`plan-${plan.id}`">
                    {{ t('subscription.selectThisPlan') }}
                  </label>
                </div>
              </div>
            </div>

            <!-- Error Message -->
            <div
              v-if="error"
              class="alert alert-danger mt-3">
              {{ error }}
            </div>
          </div>

          <!-- Downgrade Section -->
          <div
            v-if="isPremium"
            class="downgrade-section mt-4 pt-4 border-top">
            <h6 class="text-center mb-3">
              <i class="fa-solid fa-times-circle me-2 text-danger"></i>
              {{ t('subscription.cancelSubscription') }}
            </h6>

            <p class="text-muted small text-center px-3">
              {{ t('subscription.downgradeDescription') }}
            </p>

            <div class="text-center mb-3">
              <button
                class="btn btn-outline-danger"
                @click="toggleDowngradeForm"
                :disabled="isLoadingDowngrade">
                <i class="fa-solid fa-arrow-down me-2"></i>
                {{ t('subscription.downgradeToFree') }}
              </button>
            </div>

            <!-- Downgrade Form -->
            <div
              v-if="showDowngradeForm"
              class="downgrade-form mt-3 p-3 border rounded">
              <div
                v-if="isLoadingDowngrade"
                class="text-center">
                <div
                  class="spinner-border text-primary"
                  role="status">
                  <span class="visually-hidden">
                    {{ t('subscription.loadingAccounts') }}
                  </span>
                </div>
              </div>

              <div v-else>
                <!-- Message when no selection needed -->
                <div
                  v-if="!needsAnySelection"
                  class="alert alert-success mb-3">
                  <i class="fa-solid fa-check-circle me-2"></i>
                  {{ t('subscription.alreadyWithinLimits') }}
                </div>

                <!-- Accounts Selection (only if exceeds limit) -->
                <div
                  v-if="needsAccountSelection"
                  class="mb-3">
                  <label class="form-label fw-bold">
                    {{
                      t('subscription.selectAccountsForFree', {
                        count: accountLimit,
                      })
                    }}
                  </label>
                  <div class="accounts-grid">
                    <div
                      v-for="account in activeAccounts"
                      :key="account.id"
                      class="account-item"
                      :class="{
                        selected: selectedAccountIds.includes(account.id),
                      }"
                      @click="toggleAccountSelection(account.id)">
                      <input
                        type="checkbox"
                        :checked="selectedAccountIds.includes(account.id)"
                        :disabled="
                          selectedAccountIds.length >= accountLimit &&
                          !selectedAccountIds.includes(account.id)
                        " />
                      <span class="ms-2">{{ account.name }}</span>
                    </div>
                  </div>
                  <small class="text-muted">
                    {{ selectedAccountIds.length }} /
                    {{ accountLimit }} selected
                  </small>
                </div>

                <!-- Budget Selection (only if exceeds limit) -->
                <div
                  v-if="needsBudgetSelection"
                  class="mb-3">
                  <label class="form-label fw-bold">
                    {{
                      t('subscription.selectBudgetForFree', {
                        count: budgetLimit,
                      })
                    }}
                  </label>
                  <select
                    v-model="selectedBudgetId"
                    class="form-select">
                    <option :value="null">
                      {{ t('subscription.chooseBudget') }}
                    </option>
                    <option
                      v-for="budget in activeBudgets"
                      :key="budget.id"
                      :value="budget.id">
                      {{ budget.name }}
                    </option>
                  </select>
                </div>

                <!-- Confirm Button -->
                <div class="text-center">
                  <button
                    class="btn btn-danger"
                    @click="handleDowngrade"
                    :disabled="!canSubmitDowngrade || isLoadingDowngrade">
                    <span
                      v-if="isLoadingDowngrade"
                      class="spinner-border spinner-border-sm me-2"></span>
                    <i
                      v-else
                      class="fa-solid fa-check me-2"></i>
                    {{ t('subscription.confirmDowngrade') }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="modal-footer">
          <!-- Redirecting to Stripe overlay -->
          <div
            v-if="isRedirectingToStripe"
            class="redirect-overlay">
            <div
              class="spinner-border text-primary me-2"
              role="status">
              <span class="visually-hidden">{{ t('common.loading') }}</span>
            </div>
            <span>{{ t('payment.redirectingToStripe') }}</span>
          </div>

          <template v-else>
            <button
              class="btn btn-secondary"
              @click="emit('close')"
              :disabled="isLoading">
              {{ t('buttons.cancel') }}
            </button>
            <button
              class="btn btn-primary"
              @click="handleUpgrade"
              :disabled="!selectedPlanId || isLoading">
              <span
                v-if="isLoading"
                class="spinner-border spinner-border-sm me-2"></span>
              {{ t('subscription.selectThisPlan') }}
            </button>
          </template>
        </div>
      </div>
    </div>
  </div>

  <!-- Downgrade Confirmation Dialog -->
  <ConfirmDialog
    v-model="showConfirmDialog"
    :title="t('subscription.downgradeToFree')"
    :message="confirmDialogMessage || ''"
    :confirm-text="t('buttons.close')"
    :show-cancel="false"
    @confirm="handleConfirmDialogClose" />

  <!-- Plan Change Warning Dialog -->
  <ConfirmDialog
    v-model="showPlanChangeWarningDialog"
    :title="t('subscription.planChangeWarningTitle')"
    :message="planChangeWarningMessage || ''"
    :confirm-text="t('buttons.close')"
    :show-cancel="false"
    @confirm="showPlanChangeWarningDialog = false"
    @cancel="showPlanChangeWarningDialog = false" />

  <!-- Cancel Scheduled Change Confirmation Dialog -->
  <ConfirmDialog
    v-model="showCancelConfirmDialog"
    :title="t('subscription.cancelScheduledChangeTitle')"
    :message="t('subscription.cancelScheduledChangeConfirm')"
    :confirm-text="t('buttons.yes')"
    :cancel-text="t('buttons.no')"
    :show-cancel="true"
    @confirm="cancelScheduledChange"
    @cancel="showCancelConfirmDialog = false" />
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.8);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1050;
}

.modal-dialog {
  max-width: 700px;
  width: 90%;
}

.modal-content {
  background-color: white;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  overflow: hidden;
}

.modal-header {
  border-bottom: 2px solid #f0f0f0;
  padding: 1.5rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.modal-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0;
  color: white;
}

.btn-close-white {
  filter: brightness(0) invert(1);
}

.modal-body {
  padding: 2rem;
  max-height: 70vh;
  overflow-y: auto;
}

.subscription-info {
  border-bottom: 1px solid #e0e0e0;
  padding-bottom: 1.5rem;
}

.plans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
}

.plan-card {
  position: relative;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  padding: 1.5rem;
  cursor: pointer;
  transition: all 0.3s ease;
}

.plan-card:hover {
  border-color: #007bff;
  box-shadow: 0 4px 12px rgba(0, 123, 255, 0.1);
  transform: translateY(-2px);
}

.plan-card.selected {
  border-color: #007bff;
  background-color: #f0f8ff;
  box-shadow: 0 4px 16px rgba(0, 123, 255, 0.2);
}

.plan-card.featured {
  border-color: #ffd700;
  background: linear-gradient(135deg, #fff9e6 0%, #ffffff 100%);
}

.plan-card.disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background-color: #f5f5f5;
}

.plan-card.disabled:hover {
  transform: none;
  box-shadow: none;
  border-color: #e0e0e0;
}

.featured-badge {
  position: absolute;
  top: -12px;
  right: 1rem;
  background: linear-gradient(135deg, #ffd700 0%, #ffed4e 100%);
  color: #2c3e50;
  padding: 0.3rem 0.8rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(255, 215, 0, 0.3);
}

.plan-name {
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: #2c3e50;
}

.plan-price {
  margin-bottom: 1rem;
}

.price-amount {
  font-size: 2rem;
  font-weight: 700;
  color: #007bff;
}

.price-period {
  font-size: 0.9rem;
  color: #6c757d;
}

.original-price {
  font-size: 1rem;
  color: #999;
  text-decoration: line-through;
  display: block;
  margin-bottom: 0.25rem;
}

.price-amount.discounted {
  color: #28a745;
}

.credit-info {
  font-size: 0.75rem;
  color: #28a745;
  margin-top: 0.25rem;
  font-weight: 500;
}

.current-plan-badge {
  display: inline-block;
  background: #007bff;
  color: white;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.85rem;
  font-weight: 600;
}

.scheduled-plan-badge {
  display: inline-block;
  background: #ffc107;
  color: #212529;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.85rem;
  font-weight: 600;
}

.plan-description {
  font-size: 0.9rem;
  color: #6c757d;
  margin-bottom: 1rem;
  min-height: 40px;
}

.plan-features {
  margin-bottom: 1rem;
}

.feature {
  padding: 0.4rem 0;
  font-size: 0.9rem;
}

.modal-footer {
  border-top: 1px solid #e0e0e0;
  padding: 1rem 1.5rem;
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

/* Downgrade Section */
.downgrade-section {
  background-color: #fff5f5;
  padding: 1.5rem;
  border-radius: 8px;
  margin-top: 1.5rem;
}

.downgrade-form {
  background-color: white;
}

.accounts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.account-item {
  padding: 0.75rem;
  border: 2px solid #e0e0e0;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
}

.account-item:hover {
  border-color: #007bff;
  background-color: #f8f9fa;
}

.account-item.selected {
  border-color: #007bff;
  background-color: #e7f3ff;
}

.account-item input[type='checkbox'] {
  pointer-events: none;
}

.redirect-overlay {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 0.5rem;
  color: #007bff;
  font-weight: 500;
}

/* Mobile Responsive Styles */
@media (max-width: 576px) {
  .modal-backdrop {
    align-items: flex-start !important;
    padding: 0;
    overflow-y: auto;
  }

  .modal-dialog {
    width: 100% !important;
    max-width: 100% !important;
    margin: 0 !important;
    min-height: 100vh;
    transform: none !important;
  }

  .modal-content {
    border-radius: 0;
    min-height: 100vh;
    max-height: none;
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    padding: 1rem;
    flex-shrink: 0;
  }

  .modal-title {
    font-size: 1.25rem;
  }

  .modal-body {
    padding: 1rem;
    max-height: none;
    flex: 1;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }

  .modal-footer {
    padding: 0.75rem 1rem;
    flex-shrink: 0;
    position: sticky;
    bottom: 0;
    background: white;
    border-top: 1px solid #e0e0e0;
  }

  .plans-grid {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .plan-card {
    padding: 1rem;
  }

  .plan-name {
    font-size: 1.1rem;
  }

  .price-amount {
    font-size: 1.5rem;
  }

  .plan-description {
    min-height: auto;
    font-size: 0.85rem;
  }

  .feature {
    font-size: 0.85rem;
    padding: 0.25rem 0;
  }

  .subscription-info {
    padding-bottom: 1rem;
    margin-bottom: 1rem !important;
  }

  .subscription-info h6 {
    font-size: 0.95rem;
  }

  .downgrade-section {
    padding: 1rem;
    margin-top: 1rem !important;
  }

  .accounts-grid {
    grid-template-columns: 1fr;
  }

  .account-item {
    padding: 0.5rem 0.75rem;
  }

  .pending-change-notice .alert {
    font-size: 0.85rem;
  }

  .pending-change-notice p {
    margin-bottom: 0.75rem !important;
  }
}

@media (max-width: 768px) and (min-width: 577px) {
  .modal-dialog {
    width: 95%;
    max-width: 95%;
  }

  .modal-body {
    max-height: 80vh;
  }

  .plans-grid {
    grid-template-columns: 1fr;
  }
}
</style>
