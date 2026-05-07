<script setup>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { usePaymentStore } from '@/stores/payment';
import { useSubscriptionStore } from '@/stores/subscription';
import { SubscriptionService } from '@/services/subscriptionService';
import { UserService } from '@/services/users';
import { useUserStore } from '@/stores/user';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const paymentStore = usePaymentStore();
const subscriptionStore = useSubscriptionStore();
const userStore = useUserStore();

const isVerifying = ref(true);
const isSuccess = ref(false);
const errorMessage = ref(null);

onMounted(async () => {
  await verifyPayment();
});

async function verifyPayment() {
  try {
    isVerifying.value = true;
    errorMessage.value = null;

    // Get session ID from URL or localStorage
    const sessionId =
      route.query.session_id || paymentStore.getStoredSessionId();

    if (!sessionId) {
      errorMessage.value = t('payment.sessionExpired');
      isVerifying.value = false;
      return;
    }

    // Check payment status
    const status = await paymentStore.checkPaymentStatus(sessionId);

    if (status.paymentStatus === 'paid' || status.status === 'complete') {
      isSuccess.value = true;

      // Refresh subscription data
      const userService = new UserService(userStore);
      const subscriptionService = new SubscriptionService(
        subscriptionStore,
        userService,
      );
      await subscriptionService.getCurrentSubscription();

      // Clear stored session ID
      paymentStore.clearPaymentState();

      // Redirect to home after delay
      setTimeout(() => {
        router.push({ name: 'home' });
      }, 3000);
    } else {
      errorMessage.value = t('payment.error');
    }
  } catch (error) {
    console.error('[PaymentSuccessView] Verification failed:', error);
    errorMessage.value = error.message || t('payment.error');
  } finally {
    isVerifying.value = false;
  }
}

function goToHome() {
  router.push({ name: 'home' });
}

function goToSettings() {
  router.push({ name: 'settings' });
}
</script>

<template>
  <div class="payment-result-container">
    <div class="payment-card">
      <!-- Verifying -->
      <div
        v-if="isVerifying"
        class="text-center">
        <div
          class="spinner-border text-primary mb-3"
          role="status">
          <span class="visually-hidden">{{ t('common.loading') }}</span>
        </div>
        <h4>{{ t('payment.verifyingPayment') }}</h4>
      </div>

      <!-- Success -->
      <div
        v-else-if="isSuccess"
        class="text-center">
        <div class="success-icon mb-4">
          <i class="fa-solid fa-circle-check"></i>
        </div>
        <h2 class="mb-3">{{ t('payment.success') }}</h2>
        <p class="text-muted mb-4">{{ t('payment.successDescription') }}</p>
        <div class="d-flex justify-content-center gap-3">
          <button
            class="btn btn-primary"
            @click="goToHome">
            <i class="fa-solid fa-home me-2"></i>
            {{ t('navigation.home') }}
          </button>
          <button
            class="btn btn-outline-secondary"
            @click="goToSettings">
            <i class="fa-solid fa-cog me-2"></i>
            {{ t('navigation.settings') }}
          </button>
        </div>
        <p class="text-muted small mt-4">
          {{ t('payment.redirectingHome') }}
        </p>
      </div>

      <!-- Error -->
      <div
        v-else
        class="text-center">
        <div class="error-icon mb-4">
          <i class="fa-solid fa-circle-xmark"></i>
        </div>
        <h2 class="mb-3">{{ t('payment.error') }}</h2>
        <p class="text-danger mb-4">{{ errorMessage }}</p>
        <div class="d-flex justify-content-center gap-3">
          <button
            class="btn btn-primary"
            @click="goToHome">
            <i class="fa-solid fa-home me-2"></i>
            {{ t('navigation.home') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.payment-result-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 2rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.payment-card {
  background: white;
  border-radius: 16px;
  padding: 3rem;
  max-width: 500px;
  width: 100%;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.success-icon {
  font-size: 5rem;
  color: #28a745;
}

.error-icon {
  font-size: 5rem;
  color: #dc3545;
}
</style>
