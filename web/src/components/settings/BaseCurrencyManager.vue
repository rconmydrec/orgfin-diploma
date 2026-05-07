<script setup>
import { onBeforeMount, reactive } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';

const router = useRouter();
const { t } = useI18n();

const currencies = reactive([]);
const selectedCurrency = reactive({});

onBeforeMount(async () => {
  try {
    Object.assign(
      selectedCurrency,
      await Services.settingsService.getBaseCurrency(),
    );

    currencies.length = 0;
    currencies.push(...(await Services.currenciesService.getAllCurrencies()));
  } catch (e) {
    await processError(e, router);
  }
});

const apply = async () => {
  await Services.settingsService.setBaseCurrency(selectedCurrency);
};
</script>

<template>
  <div class="section-card">
    <h3>{{ t('message.selectBaseCurrency') }}</h3>

    <div class="current">
      {{ t('message.currentCurency') }}:
      <strong>{{ selectedCurrency.code }}</strong>
    </div>

    <select
      class="form-select"
      v-model="selectedCurrency.code"
      @change="
        () => {
          const c = currencies.find((cu) => cu.code === selectedCurrency.code);
          if (c) Object.assign(selectedCurrency, c);
        }
      ">
      <option
        v-for="c in currencies"
        :key="c.id"
        :value="c.code">
        {{ c.code }} ({{ c.name }})
      </option>
    </select>

    <button
      class="btn btn-primary"
      @click="apply">
      {{ t('buttons.apply') }}
    </button>
  </div>
</template>

<style scoped>
.section-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.current {
  font-size: 14px;
  color: #555;
}
.form-select {
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid #ced4da;
  border-radius: 4px;
}
</style>
