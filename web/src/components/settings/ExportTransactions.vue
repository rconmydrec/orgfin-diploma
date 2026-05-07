<script setup>
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { DateTime } from 'luxon';

import { Services } from '@/services/servicesConfig';

const { t } = useI18n();

const startDate = ref(DateTime.now().minus({ months: 1 }).toISODate());
const endDate = ref(DateTime.now().toISODate());
const delivery = ref('download');
const loading = ref(false);
const errorMessage = ref('');
const successMessage = ref('');

const validate = () => {
  if (!startDate.value || !endDate.value) {
    errorMessage.value = t('export.bothDatesRequired');
    return false;
  }
  if (startDate.value > endDate.value) {
    errorMessage.value = t('export.startBeforeEnd');
    return false;
  }
  return true;
};

const handleExport = async () => {
  errorMessage.value = '';
  successMessage.value = '';

  if (!validate()) {
    return;
  }

  loading.value = true;

  try {
    if (delivery.value === 'download') {
      await Services.exportService.downloadExport(
        startDate.value,
        endDate.value,
      );
      successMessage.value = t('export.downloadStarted');
    } else {
      await Services.exportService.emailExport(startDate.value, endDate.value);
      successMessage.value = t('export.emailSent');
    }
  } catch (e) {
    errorMessage.value = e.message || t('message.unexpectedError');
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="section-card">
    <h3>{{ t('export.title') }}</h3>
    <p class="text-muted">{{ t('export.description') }}</p>

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
      <label for="exportStartDate">{{ t('message.startDate') }}</label>
      <input
        id="exportStartDate"
        type="date"
        class="form-control"
        v-model="startDate" />
    </div>

    <div class="form-group">
      <label for="exportEndDate">{{ t('message.endDate') }}</label>
      <input
        id="exportEndDate"
        type="date"
        class="form-control"
        v-model="endDate" />
    </div>

    <div class="form-group">
      <label>{{ t('export.deliveryMethod') }}</label>
      <div class="delivery-options">
        <div class="form-check">
          <input
            id="deliveryDownload"
            type="radio"
            class="form-check-input"
            value="download"
            v-model="delivery" />
          <label
            class="form-check-label"
            for="deliveryDownload">
            <i class="fa-solid fa-download me-1"></i>
            {{ t('export.download') }}
          </label>
        </div>
        <div class="form-check">
          <input
            id="deliveryEmail"
            type="radio"
            class="form-check-input"
            value="email"
            v-model="delivery" />
          <label
            class="form-check-label"
            for="deliveryEmail">
            <i class="fa-solid fa-envelope me-1"></i>
            {{ t('export.sendToEmail') }}
          </label>
        </div>
      </div>
    </div>

    <button
      class="btn btn-primary"
      @click.prevent="handleExport"
      :disabled="loading"
      type="button">
      <span
        v-if="loading"
        class="spinner-border spinner-border-sm me-2"
        role="status"></span>
      {{ loading ? t('export.exporting') : t('export.exportButton') }}
    </button>
  </div>
</template>

<style scoped lang="scss">
@use '@/assets/settings-form';

.delivery-options {
  display: flex;
  gap: 24px;
}

.form-check-label {
  cursor: pointer;
}

.text-muted {
  font-size: 14px;
  color: #6c757d;
  margin: 0;
}
</style>
