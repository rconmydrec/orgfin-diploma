<script setup>
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { Services } from '@/services/servicesConfig';

const { t } = useI18n();

const currentPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const errorMessage = ref('');
const successMessage = ref('');
const loading = ref(false);

// Password visibility toggles
const showCurrentPassword = ref(false);
const showNewPassword = ref(false);
const showConfirmPassword = ref(false);

const validatePasswords = () => {
  if (!currentPassword.value || !newPassword.value || !confirmPassword.value) {
    errorMessage.value = t('message.fillAllFields');
    return false;
  }

  if (newPassword.value.length < 3) {
    errorMessage.value = t('message.passwordTooShort');
    return false;
  }

  if (newPassword.value !== confirmPassword.value) {
    errorMessage.value = t('message.passwordsDoNotMatch');
    return false;
  }

  if (currentPassword.value === newPassword.value) {
    errorMessage.value = t('message.newPasswordSameAsOld');
    return false;
  }

  return true;
};

const changePassword = async () => {
  errorMessage.value = '';
  successMessage.value = '';

  if (!validatePasswords()) {
    return;
  }

  loading.value = true;

  try {
    await Services.userService.changePassword(
      currentPassword.value,
      newPassword.value,
    );
    successMessage.value = t('message.passwordChangedSuccessfully');
    currentPassword.value = '';
    newPassword.value = '';
    confirmPassword.value = '';

    // Reset visibility toggles
    showCurrentPassword.value = false;
    showNewPassword.value = false;
    showConfirmPassword.value = false;
  } catch (e) {
    if (e.message === 'Current password is incorrect') {
      errorMessage.value = t('message.currentPasswordIncorrect');
    } else {
      errorMessage.value = e.message || t('message.unexpectedError');
    }
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="section-card">
    <h3>{{ t('message.changePassword') }}</h3>

    <div
      v-if="successMessage"
      class="alert alert-success"
      role="alert">
      {{ successMessage }}
    </div>

    <div
      v-if="errorMessage"
      class="alert alert-danger"
      role="alert">
      {{ errorMessage }}
    </div>

    <div class="form-group">
      <label for="currentPassword">{{ t('message.currentPassword') }}</label>
      <div class="input-with-icon">
        <input
          id="currentPassword"
          :type="showCurrentPassword ? 'text' : 'password'"
          class="form-control"
          v-model="currentPassword"
          :placeholder="t('message.enterCurrentPassword')" />
        <button
          type="button"
          class="toggle-password"
          @click.stop.prevent="showCurrentPassword = !showCurrentPassword"
          :aria-label="showCurrentPassword ? 'Hide password' : 'Show password'">
          <i
            :class="
              showCurrentPassword ? 'fa-solid fa-eye-slash' : 'fa-solid fa-eye'
            "></i>
        </button>
      </div>
    </div>

    <div class="form-group">
      <label for="newPassword">{{ t('message.newPassword') }}</label>
      <div class="input-with-icon">
        <input
          id="newPassword"
          :type="showNewPassword ? 'text' : 'password'"
          class="form-control"
          v-model="newPassword"
          :placeholder="t('message.enterNewPassword')" />
        <button
          type="button"
          class="toggle-password"
          @click.stop.prevent="showNewPassword = !showNewPassword"
          :aria-label="showNewPassword ? 'Hide password' : 'Show password'">
          <i
            :class="
              showNewPassword ? 'fa-solid fa-eye-slash' : 'fa-solid fa-eye'
            "></i>
        </button>
      </div>
    </div>

    <div class="form-group">
      <label for="confirmPassword">{{ t('message.confirmNewPassword') }}</label>
      <div class="input-with-icon">
        <input
          id="confirmPassword"
          :type="showConfirmPassword ? 'text' : 'password'"
          class="form-control"
          v-model="confirmPassword"
          :placeholder="t('message.enterConfirmPassword')" />
        <button
          type="button"
          class="toggle-password"
          @click.stop.prevent="showConfirmPassword = !showConfirmPassword"
          :aria-label="showConfirmPassword ? 'Hide password' : 'Show password'">
          <i
            :class="
              showConfirmPassword ? 'fa-solid fa-eye-slash' : 'fa-solid fa-eye'
            "></i>
        </button>
      </div>
    </div>

    <button
      class="btn btn-primary"
      @click.prevent="changePassword"
      :disabled="loading"
      type="button">
      <span
        v-if="loading"
        class="spinner-border spinner-border-sm me-2"
        role="status"></span>
      {{ loading ? t('buttons.changing') : t('buttons.changePassword') }}
    </button>
  </div>
</template>

<style scoped lang="scss">
@use '@/assets/settings-form';

.input-with-icon {
  position: relative;
  display: flex;
  align-items: center;
}

.form-control {
  padding-right: 40px;
}

.toggle-password {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  cursor: pointer;
  padding: 5px;
  color: #6c757d;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.2s;
  z-index: 10;
  pointer-events: auto;
}

.toggle-password:hover {
  color: #1e90ff;
}

.toggle-password:focus {
  outline: none;
}

.toggle-password:active {
  transform: scale(0.95);
}

.toggle-password i {
  font-size: 16px;
  pointer-events: none;
}
</style>
