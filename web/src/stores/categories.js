import { reactive, ref } from 'vue';
import { defineStore } from 'pinia';
import { DateTime } from 'luxon';

export const useCategoriesStore = defineStore('categories', () => {
  const categories = reactive([]);
  const lastUpdated = reactive(DateTime.now());
  const shouldUpdate = ref(true);

  return {
    categories,
    lastUpdated,
    shouldUpdate,
  };
});
