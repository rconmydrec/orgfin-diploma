<script setup>
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { usePaymentStore } from '@/stores/payment';

const { t } = useI18n();
const router = useRouter();
const paymentStore = usePaymentStore();

onMounted(() => {
  // Clear any stored payment state
  paymentStore.clearPaymentState();
});

function goToHome() {
  router.push({ name: 'home' });
}

function tryAgain() {
  router.push({ name: 'settings' });
}
</script>

<template>
  <div class="payment-result-container">
    <div class="payment-card">
      <div class="text-center">
        <div class="cancel-icon mb-4">
          <i class="fa-solid fa-ban"></i>
        </div>
        <h2 class="mb-3">{{ t('payment.cancelled') }}</h2>
        <p class="text-muted mb-4">{{ t('payment.cancelledDescription') }}</p>
        <div class="d-flex justify-content-center gap-3">
          <button
            class="btn btn-primary"
            @click="tryAgain">
            <i class="fa-solid fa-redo me-2"></i>
            {{ t('payment.tryAgain') }}
          </button>
          <button
            class="btn btn-outline-secondary"
            @click="goToHome">
            <i class="fa-solid fa-home me-2"></i>
            {{ t('payment.goBack') }}
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

.cancel-icon {
  font-size: 5rem;
  color: #6c757d;
}
</style>
