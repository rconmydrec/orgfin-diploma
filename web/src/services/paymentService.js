import { request } from './requests';

/**
 * Service for handling Stripe payment operations
 */
export class PaymentService {
  userService;

  constructor(userService) {
    this.userService = userService;
  }

  /**
   * Create a Stripe Checkout Session for subscription upgrade
   * @param {number} planId - The ID of the plan to subscribe to
   * @returns {Promise<{checkoutUrl: string, sessionId: string}>}
   */
  async createCheckoutSession(planId) {
    const url = '/payments/checkout';
    const response = await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify({ planId }),
      },
      { userService: this.userService },
    );

    return {
      checkoutUrl: response.checkoutUrl,
      sessionId: response.sessionId,
    };
  }

  /**
   * Create a Stripe Customer Portal session for billing management
   * @param {string|null} returnUrl - Optional URL to return to after portal
   * @returns {Promise<{portalUrl: string}>}
   */
  async createPortalSession(returnUrl = null) {
    const url = '/payments/portal';
    const body = returnUrl ? { returnUrl } : {};

    const response = await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify(body),
      },
      { userService: this.userService },
    );

    return {
      portalUrl: response.portalUrl,
    };
  }

  /**
   * Get current payment/subscription status
   * @returns {Promise<Object>}
   */
  async getPaymentStatus() {
    const url = '/payments/status';
    const response = await request(url, {}, { userService: this.userService });
    return response;
  }

  /**
   * Get status of a specific checkout session
   * @param {string} sessionId - Stripe checkout session ID
   * @returns {Promise<{sessionId: string, status: string, paymentStatus: string, customerId: string, subscriptionId: string}>}
   */
  async getCheckoutSessionStatus(sessionId) {
    const url = `/payments/session/${sessionId}`;
    const response = await request(url, {}, { userService: this.userService });
    return response;
  }

  /**
   * Get upgrade price for a plan with credit calculation
   * @param {number} planId - The ID of the plan
   * @returns {Promise<{planId: number, planName: string, originalPrice: number, creditAmount: number, creditDays: number, adjustedPrice: number, currencyCode: string}>}
   */
  async getUpgradePrice(planId) {
    const url = `/payments/upgrade-price/${planId}`;
    const response = await request(url, {}, { userService: this.userService });
    return response;
  }

  /**
   * Change subscription plan for existing Stripe subscription
   * @param {number} planId - The ID of the new plan
   * @returns {Promise<{success: boolean, isUpgrade: boolean, newPlanName: string, effectiveDate: string, message: string, scheduleId: string|null}>}
   */
  async changePlan(planId) {
    const url = '/payments/change-plan';
    const response = await request(
      url,
      {
        method: 'POST',
        body: JSON.stringify({ planId }),
      },
      { userService: this.userService },
    );
    return response;
  }

  /**
   * Cancel a scheduled plan change
   * @returns {Promise<{success: boolean, message: string}>}
   */
  async cancelScheduledChange() {
    const url = '/payments/cancel-scheduled-change';
    const response = await request(
      url,
      {
        method: 'POST',
      },
      { userService: this.userService },
    );
    return response;
  }
}
