import { request } from './requests';

export class SubscriptionService {
  subscriptionStore;
  userService;

  constructor(subscriptionStore, userService) {
    this.subscriptionStore = subscriptionStore;
    this.userService = userService;
  }

  /**
   * Get current user subscription with limits info
   * @returns {Promise<Object>} Subscription data with limits
   */
  async getCurrentSubscription() {
    const url = '/subscriptions/status';
    const subscription = await request(
      url,
      {},
      { userService: this.userService },
    );
    if (subscription) {
      this.subscriptionStore.subscription = subscription;
      this.subscriptionStore.limits = subscription.limits || {};
    }
    return subscription;
  }

  /**
   * Check subscription status (trial expired, requires downgrade)
   * @returns {Promise<Object>} Status object
   */
  async checkSubscriptionStatus() {
    const url = '/subscriptions/status';
    const status = await request(url, {}, { userService: this.userService });
    if (status) {
      this.subscriptionStore.isExpired = status.is_expired || false;
      this.subscriptionStore.requiresDowngrade =
        status.requires_downgrade || false;
    }
    return status;
  }

  /**
   * Get available subscription plans
   * @returns {Promise<Array>} Array of available plans
   */
  async getAvailablePlans() {
    const url = '/subscriptions/plans';
    const plans = await request(url, {}, { userService: this.userService });
    return plans;
  }

  /**
   * Upgrade to premium plan
   * @param {number} planId - The ID of the plan to upgrade to
   * @param {number|null} adjustedPrice - Optional adjusted price (with credit from current subscription)
   * @returns {Promise<Object>} Updated subscription
   */
  async upgradeToPremium(planId, adjustedPrice = null) {
    const url = '/subscriptions/upgrade';
    const body = { planId };
    if (adjustedPrice !== null) {
      body.adjustedPrice = adjustedPrice;
    }

    const subscription = await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify(body),
      },
      { userService: this.userService },
    );

    if (subscription) {
      this.subscriptionStore.subscription = subscription;
      this.subscriptionStore.requiresDowngrade = false;
      this.subscriptionStore.isExpired = false;
    }
    return subscription;
  }

  /**
   * Downgrade to free plan with entity selection
   * @param {Array<number>} accountIds - Array of 2 account IDs to keep active
   * @param {number} budgetId - Budget ID to keep active
   * @returns {Promise<Object>} Updated subscription
   */
  async downgradeToFree(accountIds, budgetId) {
    const url = '/subscriptions/downgrade';
    const subscription = await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify({
          accountIds,
          budgetId,
        }),
      },
      { userService: this.userService },
    );

    if (subscription) {
      this.subscriptionStore.subscription = subscription;
      this.subscriptionStore.requiresDowngrade = false;
    }
    return subscription;
  }

  /**
   * Clear subscription data from store
   */
  clearSubscription() {
    this.subscriptionStore.subscription = null;
    this.subscriptionStore.limits = {};
    this.subscriptionStore.isExpired = false;
    this.subscriptionStore.requiresDowngrade = false;
  }
}
