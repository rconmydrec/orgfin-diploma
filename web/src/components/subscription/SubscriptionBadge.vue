<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useSubscriptionStore } from '@/stores/subscription';

const { t } = useI18n();
const subscriptionStore = useSubscriptionStore();

const emit = defineEmits(['click']);

const isTrial = computed(() => subscriptionStore.isTrial);
const isPremium = computed(() => subscriptionStore.isPremium);
const trialDaysRemaining = computed(() => subscriptionStore.trialDaysRemaining);

const badgeClass = computed(() => {
  if (isPremium.value) return 'badge-premium';
  if (isTrial.value) return 'badge-trial';
  return 'badge-free';
});

const badgeIcon = computed(() => {
  if (isPremium.value) return 'fa-crown';
  if (isTrial.value) return 'fa-clock';
  return 'fa-hand-holding-heart';
});

const badgeText = computed(() => {
  if (isPremium.value) return t('subscription.premium');
  if (isTrial.value) {
    if (trialDaysRemaining.value > 0) {
      return t('subscription.trialDaysLeft', {
        days: trialDaysRemaining.value,
      });
    }
    return t('subscription.trialExpired');
  }
  return t('subscription.free');
});
</script>

<template>
  <div
    class="subscription-badge"
    :class="badgeClass"
    @click="emit('click')"
    role="button"
    tabindex="0">
    <i
      class="fa-solid me-1"
      :class="badgeIcon"></i>
    <span>{{ badgeText }}</span>
  </div>
</template>

<style scoped>
.subscription-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 0.8rem;
  border-radius: 20px;
  font-size: 0.85rem;
  font-weight: 600;
  white-space: nowrap;
  transition: all 0.3s ease;
  cursor: pointer;
}

.subscription-badge:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.badge-premium {
  background: linear-gradient(135deg, #ffd700 0%, #ffed4e 100%);
  color: #2c3e50;
  box-shadow: 0 2px 8px rgba(255, 215, 0, 0.3);
}

.badge-trial {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: white;
  box-shadow: 0 2px 8px rgba(79, 172, 254, 0.3);
}

.badge-free {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.subscription-badge i {
  font-size: 0.9rem;
}
</style>
