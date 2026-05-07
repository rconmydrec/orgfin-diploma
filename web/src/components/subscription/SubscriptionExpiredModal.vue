<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useSubscriptionStore } from '@/stores/subscription';
import { usePaymentStore } from '@/stores/payment';
import { useUserStore } from '@/stores/user';
import { SubscriptionService } from '@/services/subscriptionService';
import { UserService } from '@/services/users';

const { t } = useI18n();

const props = defineProps({
  show: {
    type: Boolean,
    required: true,
  },
  accounts: {
    type: Array,
    required: true,
  },
  budgets: {
    type: Array,
    required: true,
  },
});

const emit = defineEmits(['downgrade', 'close']);

const subscriptionStore = useSubscriptionStore();
const paymentStore = usePaymentStore();
const userStore = useUserStore();
const userService = new UserService(userStore);
const subscriptionService = new SubscriptionService(
  subscriptionStore,
  userService,
);

// State for plan selection
const availablePlans = ref([]);
const selectedPlanId = ref(null);
const isLoadingPlans = ref(false);
const isRedirectingToStripe = ref(false);
const planError = ref('');

// State for downgrade option
const showDowngradeForm = ref(false);
const selectedAccountIds = ref([]);
const selectedBudgetId = ref(null);
const isSubmitting = ref(false);
const error = ref('');

// Computed - Get limits from subscription store (loaded via API)
const accountLimit = computed(() => subscriptionStore.accountsLimit || 2);
const budgetLimit = computed(() => subscriptionStore.budgetsLimit || 1);

const activeAccounts = computed(() =>
  props.accounts.filter((acc) => !acc.isDeleted && !acc.isArchived),
);

const activeBudgets = computed(() =>
  props.budgets.filter((budget) => !budget.isDeleted),
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

const trialDaysRemaining = computed(() => subscriptionStore.trialDaysRemaining);

// Load plans when modal opens
watch(
  () => props.show,
  async (newValue) => {
    if (newValue) {
      await loadPlans();
    } else {
      // Reset state when modal closes
      selectedPlanId.value = null;
      planError.value = '';
      isRedirectingToStripe.value = false;
      showDowngradeForm.value = false;
      selectedAccountIds.value = [];
      selectedBudgetId.value = null;
      error.value = '';
    }
  },
);

async function loadPlans() {
  try {
    isLoadingPlans.value = true;
    planError.value = '';
    console.log('[SubscriptionExpiredModal] Loading plans...');
    const plans = await subscriptionService.getAvailablePlans();
    console.log('[SubscriptionExpiredModal] Loaded plans:', plans);
    // Filter out FREE and TRIAL plans - only show premium options
    availablePlans.value = (plans || []).filter(
      (plan) => plan.planType !== 'trial' && plan.planType !== 'free',
    );
    console.log(
      '[SubscriptionExpiredModal] Filtered premium plans:',
      availablePlans.value,
    );
    // Auto-select featured plan if available, otherwise select first
    const featuredPlan = availablePlans.value.find((p) => p.isFeatured);
    if (featuredPlan) {
      selectedPlanId.value = featuredPlan.id;
    } else if (availablePlans.value.length > 0) {
      selectedPlanId.value = availablePlans.value[0].id;
    }
    console.log(
      '[SubscriptionExpiredModal] Selected plan ID:',
      selectedPlanId.value,
    );
  } catch (err) {
    console.error('[SubscriptionExpiredModal] Failed to load plans:', err);
    planError.value = t('subscription.failedToLoadPlans');
  } finally {
    isLoadingPlans.value = false;
  }
}

async function handleUpgrade() {
  if (!selectedPlanId.value) {
    planError.value = t('subscription.selectPlan');
    return;
  }

  try {
    isRedirectingToStripe.value = true;
    planError.value = '';
    await paymentStore.initiateCheckout(selectedPlanId.value);
    // Note: If successful, user will be redirected to Stripe
    // This code only runs if there's an error before redirect
  } catch (err) {
    console.error('[SubscriptionExpiredModal] Checkout failed:', err);
    planError.value = err.message || t('subscription.upgradeError');
    isRedirectingToStripe.value = false;
  }
}

function formatPrice(price, currencyCode) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currencyCode || 'USD',
  }).format(price);
}

function showDowngradeSelection() {
  showDowngradeForm.value = true;
  error.value = '';
}

function handleAccountSelection(accountId) {
  const index = selectedAccountIds.value.indexOf(accountId);

  if (index > -1) {
    // Remove if already selected
    selectedAccountIds.value.splice(index, 1);
  } else {
    // Add if not selected and less than limit accounts
    if (selectedAccountIds.value.length < accountLimit.value) {
      selectedAccountIds.value.push(accountId);
    }
  }
}

function isAccountSelected(accountId) {
  return selectedAccountIds.value.includes(accountId);
}

async function handleDowngrade() {
  if (!canSubmitDowngrade.value) {
    error.value = t('subscription.selectRequiredEntities');
    return;
  }

  isSubmitting.value = true;
  error.value = '';

  try {
    await emit('downgrade', {
      // Send empty array if no account selection needed (user already within limits)
      accountIds: needsAccountSelection.value ? selectedAccountIds.value : [],
      // Send null if no budget selection needed (user already within limits)
      budgetId: needsBudgetSelection.value ? selectedBudgetId.value : null,
    });
  } catch (err) {
    error.value = err.message || t('subscription.downgradeError');
  } finally {
    isSubmitting.value = false;
  }
}

function cancelDowngrade() {
  showDowngradeForm.value = false;
  selectedAccountIds.value = [];
  selectedBudgetId.value = null;
  error.value = '';
}
</script>

<template>
  <div
    v-if="show"
    class="modal-backdrop"
    @click.self="() => {}">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content">
        <!-- Header -->
        <div class="modal-header">
          <h5 class="modal-title">
            <i
              class="fa-solid fa-crown me-2"
              style="color: #ffd700"></i>
            {{ t('subscription.trialExpiredTitle') }}
          </h5>
        </div>

        <!-- Body -->
        <div class="modal-body">
          <p class="text-center mb-4">
            {{
              t('subscription.trialExpiredMessage', {
                days: trialDaysRemaining,
              })
            }}
          </p>

          <!-- Upgrade Option with Plan Selection -->
          <div
            v-if="!showDowngradeForm"
            class="options-container">
            <div class="option-card premium-option">
              <h6>
                <i class="fa-solid fa-star text-warning me-2"></i>
                {{ t('subscription.upgradeToPremium') }}
              </h6>

              <!-- Loading Plans -->
              <div
                v-if="isLoadingPlans"
                class="text-center py-3">
                <div
                  class="spinner-border text-primary"
                  role="status">
                  <span class="visually-hidden">{{ t('common.loading') }}</span>
                </div>
              </div>

              <!-- Error loading plans -->
              <div
                v-else-if="planError"
                class="alert alert-danger mt-2">
                {{ planError }}
              </div>

              <!-- No plans available -->
              <div
                v-else-if="availablePlans.length === 0"
                class="alert alert-warning mt-2">
                {{ t('subscription.failedToLoadPlans') }}
                <button
                  class="btn btn-sm btn-outline-primary ms-2"
                  @click="loadPlans">
                  <i class="fa-solid fa-refresh me-1"></i>
                  {{ t('buttons.retry') || 'Retry' }}
                </button>
              </div>

              <!-- Plans Grid -->
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
                  }"
                  @click="selectedPlanId = plan.id">
                  <div
                    v-if="plan.isFeatured"
                    class="featured-badge">
                    <i class="fa-solid fa-star me-1"></i>
                    {{ t('subscription.mostPopular') }}
                  </div>

                  <h5 class="plan-name">
                    {{
                      plan.translationKey ? t(plan.translationKey) : plan.name
                    }}
                  </h5>

                  <div class="plan-price">
                    <span class="price-amount">
                      {{ formatPrice(plan.price, plan.currencyCode) }}
                    </span>
                    <span
                      v-if="plan.billingPeriod"
                      class="price-period">
                      / {{ plan.billingPeriod.name }}
                    </span>
                  </div>

                  <div class="form-check mt-2">
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

              <!-- Premium Features List -->
              <ul class="feature-list mt-3">
                <li>
                  <i class="fa-solid fa-check text-success me-2"></i>
                  {{ t('subscription.unlimitedAccounts') }}
                </li>
                <li>
                  <i class="fa-solid fa-check text-success me-2"></i>
                  {{ t('subscription.unlimitedBudgets') }}
                </li>
                <li>
                  <i class="fa-solid fa-check text-success me-2"></i>
                  {{ t('subscription.unlimitedPlanning') }}
                </li>
              </ul>

              <!-- Upgrade Button -->
              <button
                class="btn btn-primary btn-lg w-100 mt-3"
                @click="handleUpgrade"
                :disabled="
                  !selectedPlanId || isRedirectingToStripe || isLoadingPlans
                ">
                <span
                  v-if="isRedirectingToStripe"
                  class="d-flex align-items-center justify-content-center">
                  <span
                    class="spinner-border spinner-border-sm me-2"
                    role="status"></span>
                  {{ t('payment.redirectingToStripe') }}
                </span>
                <span v-else>
                  <i class="fa-solid fa-arrow-up me-2"></i>
                  {{ t('subscription.proceedToPayment') }}
                </span>
              </button>
            </div>

            <div class="divider">
              <span>{{ t('subscription.or') }}</span>
            </div>

            <div class="option-card free-option">
              <h6>
                <i class="fa-solid fa-hand-holding-heart text-info me-2"></i>
                {{ t('subscription.continueWithFree') }}
              </h6>
              <ul class="feature-list">
                <li>
                  <i class="fa-solid fa-info-circle text-info me-2"></i>
                  {{
                    t('subscription.twoAccountsMax', { count: accountLimit })
                  }}
                </li>
                <li>
                  <i class="fa-solid fa-info-circle text-info me-2"></i>
                  {{ t('subscription.oneBudgetMax', { count: budgetLimit }) }}
                </li>
                <li>
                  <i class="fa-solid fa-info-circle text-info me-2"></i>
                  {{ t('subscription.planningLimit14Days') }}
                </li>
              </ul>
              <button
                class="btn btn-outline-secondary btn-lg w-100 mt-3"
                @click="showDowngradeSelection">
                <i class="fa-solid fa-arrow-down me-2"></i>
                {{ t('subscription.selectEntities') }}
              </button>
            </div>
          </div>

          <!-- Downgrade Form -->
          <div
            v-else
            class="downgrade-form">
            <button
              class="btn btn-sm btn-link mb-3"
              @click="cancelDowngrade">
              <i class="fa-solid fa-arrow-left me-1"></i>
              {{ t('buttons.back') }}
            </button>

            <!-- Message when no selection needed -->
            <div
              v-if="!needsAnySelection"
              class="alert alert-success mb-4">
              <i class="fa-solid fa-check-circle me-2"></i>
              {{ t('subscription.alreadyWithinLimits') }}
            </div>

            <h6
              v-if="needsAnySelection"
              class="mb-3">
              {{ t('subscription.selectAccountsAndBudget') }}
            </h6>

            <!-- Select Accounts (only if exceeds limit) -->
            <div
              v-if="needsAccountSelection"
              class="form-group mb-4">
              <label class="form-label">
                {{
                  t('subscription.selectTwoAccounts', { count: accountLimit })
                }}
                <span class="badge bg-secondary ms-2">
                  {{ selectedAccountIds.length }}/{{ accountLimit }}
                </span>
              </label>
              <div class="entity-list">
                <div
                  v-for="account in activeAccounts"
                  :key="account.id"
                  class="entity-item"
                  :class="{
                    selected: isAccountSelected(account.id),
                    disabled:
                      selectedAccountIds.length >= accountLimit &&
                      !isAccountSelected(account.id),
                  }"
                  @click="handleAccountSelection(account.id)">
                  <div class="form-check">
                    <input
                      class="form-check-input"
                      type="checkbox"
                      :checked="isAccountSelected(account.id)"
                      :disabled="
                        selectedAccountIds.length >= accountLimit &&
                        !isAccountSelected(account.id)
                      " />
                    <label class="form-check-label">
                      {{ account.name }}
                      <span class="text-muted ms-2">
                        ({{ account.balance }} {{ account.currency?.code }})
                      </span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <!-- Select Budget (only if exceeds limit) -->
            <div
              v-if="needsBudgetSelection"
              class="form-group mb-4">
              <label class="form-label">
                {{ t('subscription.selectOneBudget', { count: budgetLimit }) }}
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
                  {{ budget.name || budget.category?.name }} ({{
                    budget.amount
                  }}
                  {{ budget.currency?.code }})
                </option>
              </select>
            </div>

            <!-- Error Message -->
            <div
              v-if="error"
              class="alert alert-danger">
              {{ error }}
            </div>

            <!-- Actions -->
            <div class="d-flex gap-2">
              <button
                class="btn btn-secondary flex-fill"
                @click="cancelDowngrade"
                :disabled="isSubmitting">
                {{ t('buttons.cancel') }}
              </button>
              <button
                class="btn btn-primary flex-fill"
                @click="handleDowngrade"
                :disabled="!canSubmitDowngrade || isSubmitting">
                <span
                  v-if="isSubmitting"
                  class="spinner-border spinner-border-sm me-2"></span>
                {{ t('subscription.confirmDowngrade') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.95);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1050;
}

.modal-dialog {
  max-width: 650px;
  width: 90%;
}

.modal-content {
  background-color: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(10px);
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  border: 2px solid rgba(103, 126, 234, 0.3);
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

.modal-body {
  padding: 2rem;
  max-height: 80vh;
  overflow-y: auto;
}

.options-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.option-card {
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  padding: 1.5rem;
  transition: all 0.3s ease;
}

.option-card:hover {
  border-color: #007bff;
  box-shadow: 0 4px 12px rgba(0, 123, 255, 0.1);
}

.premium-option {
  background: linear-gradient(135deg, #fff9e6 0%, #ffffff 100%);
}

.free-option {
  background: linear-gradient(135deg, #f0f8ff 0%, #ffffff 100%);
}

.feature-list {
  list-style: none;
  padding: 0;
  margin: 1rem 0;
}

.feature-list li {
  padding: 0.5rem 0;
}

/* Plans Grid */
.plans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin: 1rem 0;
}

.plan-card {
  position: relative;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  padding: 1rem;
  cursor: pointer;
  transition: all 0.3s ease;
  background: white;
}

.plan-card:hover {
  border-color: #007bff;
  box-shadow: 0 4px 12px rgba(0, 123, 255, 0.15);
  transform: translateY(-2px);
}

.plan-card.selected {
  border-color: #007bff;
  background-color: #f0f8ff;
  box-shadow: 0 4px 16px rgba(0, 123, 255, 0.2);
}

.plan-card.featured {
  border-color: #ffd700;
}

.plan-card.featured.selected {
  border-color: #007bff;
  background: linear-gradient(135deg, #f0f8ff 0%, #fff9e6 100%);
}

.featured-badge {
  position: absolute;
  top: -10px;
  right: 10px;
  background: linear-gradient(135deg, #ffd700 0%, #ffed4e 100%);
  color: #2c3e50;
  padding: 0.2rem 0.6rem;
  border-radius: 12px;
  font-size: 0.7rem;
  font-weight: 600;
  box-shadow: 0 2px 6px rgba(255, 215, 0, 0.3);
}

.plan-name {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: #2c3e50;
}

.plan-price {
  margin-bottom: 0.5rem;
}

.price-amount {
  font-size: 1.5rem;
  font-weight: 700;
  color: #007bff;
}

.price-period {
  font-size: 0.85rem;
  color: #6c757d;
}

.divider {
  text-align: center;
  position: relative;
  margin: 1rem 0;
}

.divider::before,
.divider::after {
  content: '';
  position: absolute;
  top: 50%;
  width: 40%;
  height: 1px;
  background-color: #ddd;
}

.divider::before {
  left: 0;
}

.divider::after {
  right: 0;
}

.divider span {
  background-color: white;
  padding: 0 1rem;
  color: #999;
  font-weight: 500;
}

.downgrade-form {
  max-height: 60vh;
  overflow-y: auto;
}

.entity-list {
  max-height: 250px;
  overflow-y: auto;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  padding: 0.5rem;
}

.entity-item {
  padding: 0.75rem;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.entity-item:hover:not(.disabled) {
  background-color: #f8f9fa;
}

.entity-item.selected {
  background-color: #e7f3ff;
  border: 1px solid #007bff;
}

.entity-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.form-check-input {
  cursor: pointer;
}

.form-check-label {
  cursor: pointer;
  user-select: none;
}
</style>
