<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import ModalWindow from '@/components/utils/ModalWindow.vue';
import { useCategoriesStore } from '@/stores/categories';
import { useAccountStore } from '@/stores/account';
import { Services } from '@/services/servicesConfig';

const { t } = useI18n();
const categoriesStore = useCategoriesStore();
const accountStore = useAccountStore();

const props = defineProps({
  template: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(['close', 'save']);

const form = ref({
  id: null,
  label: '',
  categoryId: null,
  targetAccountId: null,
});

const isTransferTemplate = computed(() => {
  return props.template?.targetAccountId != null;
});

onMounted(async () => {
  if (categoriesStore.categories.length === 0) {
    await Services.categoriesService.getUserCategories(true);
  }
  if (accountStore.accounts.length === 0) {
    await Services.accountsService.getUserAccounts({
      shouldUpdate: true,
      includeHidden: true,
    });
  }
});

watch(
  () => props.template,
  (newTemplate) => {
    if (newTemplate) {
      form.value = {
        id: newTemplate.id,
        label: newTemplate.label,
        categoryId: newTemplate.categoryId || newTemplate.category_id || null,
        targetAccountId: newTemplate.targetAccountId || null,
      };
    }
  },
  { immediate: true },
);

function handleSave() {
  emit('save', form.value);
}

function handleClose() {
  emit('close');
}
</script>

<template>
  <ModalWindow
    v-if="template"
    :close-modal="handleClose"
    :show-close-button="false">
    <template #header>
      <h5 class="modal-title">{{ t('message.editTemplate') }}</h5>
    </template>

    <template #main>
      <form @submit.prevent="handleSave">
        <!-- Label -->
        <div class="mb-3">
          <label class="form-label">
            {{ t('message.label') }}
            <span class="text-danger">*</span>
          </label>
          <input
            v-model="form.label"
            type="text"
            class="form-control"
            maxlength="50"
            required />
        </div>

        <!-- Account selector for transfer templates -->
        <div
          v-if="isTransferTemplate"
          class="mb-3">
          <label class="form-label">
            {{ t('message.targetAccount') }}
          </label>
          <select
            v-model="form.targetAccountId"
            class="form-select">
            <option
              :value="null"
              disabled>
              {{ t('message.selectAccount') }}
            </option>
            <option
              v-for="account in accountStore.accounts"
              :key="account.id"
              :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </div>

        <!-- Category selector for regular templates -->
        <div
          v-else
          class="mb-3">
          <label class="form-label">
            {{ t('message.category') }}
            <span class="text-danger">*</span>
          </label>
          <select
            v-model="form.categoryId"
            class="form-select"
            required>
            <option
              :value="null"
              disabled>
              {{ t('message.selectCategory') }}
            </option>
            <option
              v-for="category in categoriesStore.categories"
              :key="category.id"
              :value="category.id">
              {{ category.name }}
            </option>
          </select>
        </div>

        <div class="d-flex gap-2 justify-content-end">
          <button
            type="button"
            class="btn btn-secondary"
            @click="handleClose">
            {{ t('buttons.cancel') }}
          </button>
          <button
            type="submit"
            class="btn btn-primary">
            {{ t('buttons.save') }}
          </button>
        </div>
      </form>
    </template>
  </ModalWindow>
</template>

<style scoped>
.modal-title {
  margin: 0;
}
</style>
