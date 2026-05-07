<script setup>
import { onBeforeMount, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';

const router = useRouter();
const { locale, t } = useI18n();

const languages = reactive([]);
const selectedLanguageId = ref('');

onBeforeMount(async () => {
  try {
    languages.length = 0;
    languages.push(...(await Services.settingsService.getLanguages()));
    selectedLanguageId.value =
      languages.find((l) => l.code === locale.value)?.id || '';
  } catch (e) {
    await processError(e, router);
  }
});

const apply = async () => {
  const lang = languages.find((l) => l.id === selectedLanguageId.value);
  if (!lang) return;
  locale.value = lang.code;
  Services.userService.userStore.settings.language = lang.code;
  try {
    await Services.settingsService.saveUserSettings();
  } catch (e) {
    await processError(e, router);
  }
};
</script>

<template>
  <div class="section-card">
    <h3>{{ t('message.select_language') }}</h3>

    <select
      v-model="selectedLanguageId"
      class="form-select">
      <option
        v-for="l in languages"
        :key="l.id"
        :value="l.id">
        {{ l.name }}
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
.form-select {
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid #ced4da;
  border-radius: 4px;
}
</style>
