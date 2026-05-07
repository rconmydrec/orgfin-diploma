<script setup>
defineProps({
  loading: {
    type: Boolean,
    default: false,
  },
  error: {
    type: String,
    default: null,
  },
  loadingText: {
    type: String,
    default: null, // Will use i18n 'common.loading' if not provided
  },
});

const emit = defineEmits(['clearError']);

function handleClearError() {
  emit('clearError');
}
</script>

<template>
  <div class="loading-error-wrapper">
    <!-- Loading state -->
    <div
      v-if="loading"
      class="text-center py-5">
      <div
        class="spinner-border"
        role="status">
        <span class="visually-hidden">
          {{ loadingText || $t('common.loading') }}
        </span>
      </div>
    </div>

    <!-- Error state -->
    <div
      v-else-if="error"
      class="alert alert-danger alert-dismissible fade show">
      <i class="bi bi-exclamation-triangle"></i>
      {{ error }}
      <button
        type="button"
        class="btn-close"
        @click="handleClearError"></button>
    </div>

    <!-- Content slot -->
    <slot v-else></slot>
  </div>
</template>

<style scoped>
.alert {
  border-radius: 12px;
}
</style>
