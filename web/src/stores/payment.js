import { defineStore } from 'pinia';
import { ref } from 'vue';
import { PaymentService } from '@/services/paymentService';
import { useUserStore } from '@/stores/user';
import { UserService } from '@/services/users';

export const usePaymentStore = defineStore('payment', () => {
  // State
  const checkoutSessionId = ref(null);
  const isProcessingPayment = ref(false);
  const paymentError = ref(null);
  const lastPaymentStatus = ref(null);

  // Get services
  function getPaymentService() {
    const userStore = useUserStore();
    const userService = new UserService(userStore);
    return new PaymentService(userService);
  }

  /**
   * Initiate checkout - creates session and redirects to Stripe
   * @param {number} planId - The plan ID to subscribe to
   * @returns {Promise<void>}
   */
  async function initiateCheckout(planId) {
    const paymentService = getPaymentService();

    try {
      isProcessingPayment.value = true;
      paymentError.value = null;

      const { checkoutUrl, sessionId } =
        await paymentService.createCheckoutSession(planId);

      checkoutSessionId.value = sessionId;

      // Store session ID in localStorage for verification after redirect
      localStorage.setItem('stripe_checkout_session_id', sessionId);

      // Redirect to Stripe Checkout
      window.location.href = checkoutUrl;
    } catch (error) {
      console.error('[PaymentStore] Checkout initiation failed:', error);
      paymentError.value = error.message || 'Failed to initiate checkout';
      throw error;
    } finally {
      isProcessingPayment.value = false;
    }
  }

  /**
   * Check payment status for a checkout session
   * @param {string} sessionId - Stripe checkout session ID
   * @returns {Promise<Object>}
   */
  async function checkPaymentStatus(sessionId) {
    const paymentService = getPaymentService();

    try {
      isProcessingPayment.value = true;
      paymentError.value = null;

      const status = await paymentService.getCheckoutSessionStatus(sessionId);
      lastPaymentStatus.value = status;

      return status;
    } catch (error) {
      console.error('[PaymentStore] Payment status check failed:', error);
      paymentError.value = error.message || 'Failed to verify payment';
      throw error;
    } finally {
      isProcessingPayment.value = false;
    }
  }

  /**
   * Open Stripe Customer Portal for billing management
   * @param {string|null} returnUrl - Optional return URL
   * @returns {Promise<void>}
   */
  async function openCustomerPortal(returnUrl = null) {
    const paymentService = getPaymentService();

    try {
      isProcessingPayment.value = true;
      paymentError.value = null;

      const { portalUrl } = await paymentService.createPortalSession(returnUrl);

      // Redirect to Stripe Customer Portal
      window.location.href = portalUrl;
    } catch (error) {
      console.error('[PaymentStore] Customer portal opening failed:', error);
      paymentError.value = error.message || 'Failed to open billing portal';
      throw error;
    } finally {
      isProcessingPayment.value = false;
    }
  }

  /**
   * Get upgrade price for a plan
   * @param {number} planId - The plan ID
   * @returns {Promise<Object>}
   */
  async function getUpgradePrice(planId) {
    const paymentService = getPaymentService();

    try {
      return await paymentService.getUpgradePrice(planId);
    } catch (error) {
      console.error('[PaymentStore] Get upgrade price failed:', error);
      throw error;
    }
  }

  /**
   * Change subscription plan (for existing Stripe subscriptions)
   * @param {number} planId - The new plan ID
   * @returns {Promise<Object>} - Result with isUpgrade, effectiveDate, message, scheduleId
   */
  async function changePlan(planId) {
    const paymentService = getPaymentService();

    try {
      isProcessingPayment.value = true;
      paymentError.value = null;

      const result = await paymentService.changePlan(planId);
      return result;
    } catch (error) {
      console.error('[PaymentStore] Change plan failed:', error);
      paymentError.value = error.message || 'Failed to change plan';
      throw error;
    } finally {
      isProcessingPayment.value = false;
    }
  }

  /**
   * Cancel a scheduled plan change
   * @returns {Promise<Object>} - Result with success, message
   */
  async function cancelScheduledChange() {
    const paymentService = getPaymentService();

    try {
      isProcessingPayment.value = true;
      paymentError.value = null;

      const result = await paymentService.cancelScheduledChange();
      return result;
    } catch (error) {
      console.error('[PaymentStore] Cancel scheduled change failed:', error);
      paymentError.value = error.message || 'Failed to cancel scheduled change';
      throw error;
    } finally {
      isProcessingPayment.value = false;
    }
  }

  /**
   * Clear payment state
   */
  function clearPaymentState() {
    checkoutSessionId.value = null;
    isProcessingPayment.value = false;
    paymentError.value = null;
    lastPaymentStatus.value = null;
    localStorage.removeItem('stripe_checkout_session_id');
  }

  /**
   * Get stored checkout session ID from localStorage
   * @returns {string|null}
   */
  function getStoredSessionId() {
    return localStorage.getItem('stripe_checkout_session_id');
  }

  return {
    // State
    checkoutSessionId,
    isProcessingPayment,
    paymentError,
    lastPaymentStatus,

    // Actions
    initiateCheckout,
    checkPaymentStatus,
    openCustomerPortal,
    getUpgradePrice,
    changePlan,
    cancelScheduledChange,
    clearPaymentState,
    getStoredSessionId,
  };
});
