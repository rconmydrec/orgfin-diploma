import { defineStore } from 'pinia';
import { ref, reactive, computed } from 'vue';

export const useSubscriptionStore = defineStore('subscription', () => {
  // State
  const subscription = ref(null);
  const limits = reactive({});
  const isExpired = ref(false);
  const requiresDowngrade = ref(false);

  // Getters
  const planType = computed(() => subscription.value?.planType || 'free');
  const isTrial = computed(() => planType.value === 'trial');
  const isPremium = computed(() => planType.value === 'premium');
  const isFree = computed(() => planType.value === 'free');

  const trialEndsAt = computed(() => subscription.value?.trialEndsAt);
  const trialDaysRemaining = computed(() => {
    if (!isTrial.value || !trialEndsAt.value) return 0;
    const now = new Date();
    const endsAt = new Date(trialEndsAt.value);
    const diffTime = endsAt - now;
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    return Math.max(0, diffDays);
  });

  const hasAccountLimit = computed(
    () => limits.accounts !== undefined && limits.accounts !== null,
  );
  const hasBudgetLimit = computed(
    () => limits.budgets !== undefined && limits.budgets !== null,
  );
  const hasPlanningDaysLimit = computed(
    () => limits.planningDays !== undefined && limits.planningDays !== null,
  );

  const accountsLimit = computed(() => limits.accounts || null);
  const budgetsLimit = computed(() => limits.budgets || null);
  const planningDaysLimit = computed(() => limits.planningDays || null);

  // Actions would be handled by the service, but we can add helper methods here if needed
  function clearSubscription() {
    subscription.value = null;
    Object.keys(limits).forEach((key) => delete limits[key]);
    isExpired.value = false;
    requiresDowngrade.value = false;
  }

  return {
    // State
    subscription,
    limits,
    isExpired,
    requiresDowngrade,

    // Getters
    planType,
    isTrial,
    isPremium,
    isFree,
    trialEndsAt,
    trialDaysRemaining,
    hasAccountLimit,
    hasBudgetLimit,
    hasPlanningDaysLimit,
    accountsLimit,
    budgetsLimit,
    planningDaysLimit,

    // Actions
    clearSubscription,
  };
});
