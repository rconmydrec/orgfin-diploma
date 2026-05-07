<script setup>
import { ref } from 'vue';

import LanguageSelector from './LanguageSelector.vue';
import BaseCurrencyManager from './BaseCurrencyManager.vue';
import ChangePassword from './ChangePassword.vue';
import BillingManager from './BillingManager.vue';

const tab = ref('lang'); // 'lang' | 'curr' | 'password' | 'billing'
</script>

<template>
  <div class="profile-wrapper">
    <!-- Profile local menu -->
    <nav class="profile-tabs">
      <button
        class="btn btn-outline-primary"
        :class="{ active: tab === 'lang' }"
        @click="tab = 'lang'">
        <i class="fa-solid fa-language"></i>
        <span>{{ $t('buttons.language') }}</span>
      </button>

      <button
        class="btn btn-outline-primary"
        :class="{ active: tab === 'curr' }"
        @click="tab = 'curr'">
        <i class="fa-solid fa-money-bill-wave"></i>
        <span>{{ $t('buttons.baseCurrency') }}</span>
      </button>

      <button
        class="btn btn-outline-primary"
        :class="{ active: tab === 'password' }"
        @click="tab = 'password'">
        <i class="fa-solid fa-lock"></i>
        <span>{{ $t('buttons.changePassword') }}</span>
      </button>

      <button
        class="btn btn-outline-primary"
        :class="{ active: tab === 'billing' }"
        @click="tab = 'billing'">
        <i class="fa-solid fa-credit-card"></i>
        <span>{{ $t('payment.manageBilling') }}</span>
      </button>
    </nav>

    <!-- Active section -->
    <div class="profile-body">
      <LanguageSelector v-if="tab === 'lang'" />
      <BaseCurrencyManager v-else-if="tab === 'curr'" />
      <ChangePassword v-else-if="tab === 'password'" />
      <BillingManager v-else-if="tab === 'billing'" />
    </div>
  </div>
</template>

<style scoped>
.profile-wrapper {
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.profile-tabs {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}
.profile-tabs .btn {
  display: flex;
  align-items: center;
  gap: 8px;
}
.profile-tabs .btn.active,
.profile-tabs .btn.active:hover {
  background: #1e90ff;
  color: #fff;
}
.profile-body {
  width: 100%;
}

/* Mobile: icons only */
@media (max-width: 768px) {
  .profile-tabs {
    gap: 8px;
  }
  .profile-tabs .btn {
    padding: 10px 12px;
  }
  .profile-tabs .btn span {
    display: none;
  }
  .profile-tabs .btn i {
    font-size: 18px;
  }
}
</style>
