import { reactive, ref } from 'vue';
import { defineStore } from 'pinia';
import { DateTime } from 'luxon';

export const useAccountStore = defineStore('account', () => {
  const accounts = reactive([]);
  const lastUpdated = reactive(DateTime.now());
  const shouldUpdate = ref(true);
  const accountTypes = reactive([]);

  return {
    accounts,
    lastUpdated,
    shouldUpdate,
    accountTypes,
  };
});
