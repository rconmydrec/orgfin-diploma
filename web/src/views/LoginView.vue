<script setup>
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { Services } from '@/services/servicesConfig';
import { HttpError } from '@/errors/HttpError';
import { useUserStore } from '@/stores/user';
import { GoogleLogin } from 'vue3-google-login';
import UnauthHeader from '@/components/UnauthHeader.vue';

const userStore = useUserStore();
const router = useRouter();
const { t, locale } = useI18n();

const email = ref('');
const password = ref('');
const loginError = ref('');
const emailRef = ref(null);
const showPassword = ref(false);

onMounted(() => {
  emailRef.value?.focus();
});

async function submitLogin() {
  loginError.value = '';
  try {
    await Services.userService.loginUser(email.value, password.value);
    await afterLogin();
  } catch (err) {
    processError(err);
  }
}

async function afterLogin() {
  email.value = '';
  password.value = '';
  locale.value = userStore.settings.language;
  await router.push({ name: 'accounts' });
}

function processError(err) {
  if (err instanceof HttpError && err.statusCode === 401) {
    loginError.value =
      err.message === 'User not activated'
        ? t('message.userNotActivated')
        : t('message.invalidCredentials');
  } else {
    console.error(err);
    loginError.value = t('message.unexpectedError');
  }
}

const callback = async (response) => {
  try {
    await Services.userService.oauthLogin(response.credential);
    await afterLogin();
  } catch (err) {
    processError(err);
  }
};

function togglePasswordVisibility() {
  showPassword.value = !showPassword.value;
}
</script>

<template>
  <div class="login-page">
    <UnauthHeader />

    <div class="box">
      <form
        class="left"
        @submit.prevent="submitLogin"
        autocomplete="on">
        <small>{{ t('message.loginToAccount') }}</small>

        <div class="field">
          <span><i class="fa fa-envelope"></i></span>
          <input
            ref="emailRef"
            v-model="email"
            type="email"
            :placeholder="t('message.enterEmail')"
            required />
        </div>

        <div class="field password-field">
          <span><i class="fa fa-lock"></i></span>
          <input
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            :placeholder="t('message.enterPassword')"
            required />
          <button
            type="button"
            class="toggle-password"
            @click="togglePasswordVisibility"
            :title="
              showPassword
                ? t('message.hidePassword')
                : t('message.showPassword')
            ">
            <i :class="showPassword ? 'fa fa-eye-slash' : 'fa fa-eye'"></i>
          </button>
        </div>

        <button type="submit">{{ t('menu.login') }}</button>

        <div class="oauth-container">
          <GoogleLogin :callback="callback" />
        </div>

        <div
          v-if="loginError"
          class="alert">
          <strong>{{ t('message.error') }}</strong>
          {{ loginError }}
        </div>
      </form>

      <div class="right">
        <h2>{{ t('message.noAccount') }}</h2>
        <p>{{ t('message.registerInvite') }}</p>
        <RouterLink
          class="register-link"
          :to="{ name: 'register' }">
          {{ t('menu.register') }}
        </RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.password-field {
  position: relative;
}

.toggle-password {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.3s;
}

.toggle-password:hover {
  color: #333;
}

.toggle-password i {
  font-size: 16px;
}
</style>
