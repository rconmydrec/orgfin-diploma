import { defineStore } from 'pinia';
import { ref, reactive } from 'vue';
import router from '@/router';

export const useUserStore = defineStore('user', () => {
  const user = reactive({});
  const accessToken = ref(null);
  const isLoggedIn = ref(false);

  const currencies = reactive([]);
  const categories = reactive([]);
  const settings = reactive({});
  const baseCurrency = ref('');
  const transactionTemplates = reactive([]);

  const timeLeft = ref('00:00');
  let timer = null;

  const updateTimeLeft = async () => {
    if (!accessToken.value) {
      timeLeft.value = '00:00';
      clearInterval(timer);
      return;
    }

    const base64Url = accessToken.value.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = atob(base64);
    const decoded = JSON.parse(jsonPayload);

    const currentTime = Math.floor(Date.now() / 1000);
    const timeRemaining = decoded.exp - currentTime;

    if (timeRemaining <= 0) {
      timeLeft.value = '00:00';
      clearInterval(timer);
      isLoggedIn.value = false;
      accessToken.value = null;
      localStorage.clear();
      await router.push({ name: 'login' });
    } else {
      const minutes = Math.floor(timeRemaining / 60);
      const seconds = timeRemaining % 60;
      timeLeft.value = `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
    }
  };

  const startTimer = () => {
    if (timer) clearInterval(timer);
    updateTimeLeft();
    timer = setInterval(updateTimeLeft, 1000);
  };

  const stopTimer = () => {
    if (timer) clearInterval(timer);
    timer = null;
    timeLeft.value = '00:00';
  };

  if (localStorage.getItem('accessToken')) {
    accessToken.value = localStorage.getItem('accessToken');
    isLoggedIn.value = localStorage.getItem('isLoggedIn') === 'true';
    try {
      Object.assign(user, JSON.parse(localStorage.getItem('user')));
      Object.assign(
        settings,
        JSON.parse(localStorage.getItem('settings') || '{}'),
      );
      baseCurrency.value = localStorage.getItem('baseCurrency') || '';
    } catch (e) {
      console.warn('Failed to hydrate user store:', e);
    }
    startTimer();
  }

  return {
    user,
    accessToken,
    isLoggedIn,
    currencies,
    categories,
    settings,
    baseCurrency,
    timeLeft,
    startTimer,
    stopTimer,
    transactionTemplates,
  };
});
