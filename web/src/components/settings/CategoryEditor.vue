<script setup>
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Services } from '@/services/servicesConfig';
import ModalWindow from '@/components/utils/ModalWindow.vue';

const props = defineProps({
  closeModal: Function,
  category: Object,
  categories: Array,
  allCategories: Object,
});
const emit = defineEmits(['categoryUpdated']);
const { t } = useI18n();

const state = reactive({
  id: props.category.id,
  name: props.category.name || '',
  isIncome: props.category.isIncome || false,
  isDeleted: props.category.isDeleted || false,
  parentId: props.category.parentId || '',
});
const askDelete = ref(false);

const categoryOptions = computed(() => {
  const pool = state.isIncome
    ? props.allCategories?.income || props.categories
    : props.allCategories?.expenses || props.categories;
  return pool.map((c) => ({ id: c.id, name: c.name }));
});

function toggleType(isIncome) {
  state.isIncome = isIncome;
  // Reset parent category when switching type if current parent doesn't exist in new pool
  if (state.parentId) {
    const existsInPool = categoryOptions.value.some(
      (c) => c.id === state.parentId,
    );
    if (!existsInPool) {
      state.parentId = '';
    }
  }
}

async function save() {
  const dto = {
    id: state.id,
    name: state.name,
    isIncome: state.isIncome,
    isDeleted: state.isDeleted,
    parentId: state.parentId || null,
  };
  state.id
    ? await Services.categoriesService.updateCategory(dto)
    : await Services.categoriesService.createCategory(dto);
  emit('categoryUpdated');
  props.closeModal();
}

function promptDelete() {
  askDelete.value = true;
}

async function reallyDelete() {
  await Services.categoriesService.deleteCategory(state.id);
  emit('categoryUpdated');
  props.closeModal();
}
</script>

<template>
  <ModalWindow :close-modal="props.closeModal">
    <template #header>
      <div class="ed-header">
        <div class="title">
          <i
            :class="[
              'fa-solid',
              state.isIncome ? 'fa-circle-up income' : 'fa-circle-down expense',
            ]" />
          <h2>{{ t('message.category') }}</h2>
          <span
            class="pill"
            :class="state.isIncome ? 'income' : 'expense'">
            {{ state.isIncome ? t('message.income') : t('message.expense') }}
          </span>
          <span
            v-if="state.id"
            class="muted">
            #{{ state.id }}
          </span>
        </div>

        <button
          class="btn btn-outline-primary"
          @click="props.closeModal">
          {{ t('buttons.cancel') }}
        </button>
      </div>
    </template>

    <template #main>
      <div class="ed-grid">
        <div class="col main">
          <label class="field">
            <span class="label">{{ t('message.name') }}</span>
            <input
              v-model="state.name"
              type="text"
              class="form-input"
              placeholder="…" />
          </label>

          <div class="field">
            <span class="label">{{ t('message.type') }}</span>
            <div class="segmented">
              <button
                type="button"
                class="btn btn-outline-primary"
                :class="{ active: state.isIncome }"
                @click="toggleType(true)">
                <i class="fa-solid fa-circle-up"></i>
                {{ t('message.income') }}
              </button>
              <button
                type="button"
                class="btn btn-outline-primary"
                :class="{ active: !state.isIncome }"
                @click="toggleType(false)">
                <i class="fa-solid fa-circle-down"></i>
                {{ t('message.expense') }}
              </button>
            </div>
          </div>

          <label
            v-if="!props.category.children?.length"
            class="field">
            <span class="label">{{ t('message.group') }}</span>
            <select
              v-model="state.parentId"
              class="select-input">
              <option value="">{{ t('message.none') }}</option>
              <option
                v-for="opt in categoryOptions"
                :key="opt.id"
                :value="opt.id">
                {{ opt.name }}
              </option>
            </select>
          </label>

          <label class="field switch">
            <input
              type="checkbox"
              v-model="state.isDeleted" />
            <span>{{ t('message.isDeleted') }}</span>
          </label>
        </div>

        <div class="col side">
          <div class="hint-card">
            <p class="hint-title">{{ t('message.tips') ?? 'Tips' }}</p>
            <ul>
              <li v-if="props.category.children?.length">
                {{
                  t('message.hasSubcategories') ??
                  'This group has subcategories; parent cannot be changed.'
                }}
              </li>
              <li>
                {{
                  t('message.youCanReassign') ??
                  'You can switch Income/Expense at any time.'
                }}
              </li>
              <li>
                {{
                  t('message.archiveHint') ??
                  'Archiving hides the category from pickers but keeps history.'
                }}
              </li>
            </ul>
          </div>
        </div>
      </div>

      <div class="actions">
        <button
          class="btn btn-primary"
          @click="save">
          {{ t('buttons.save') }}
        </button>

        <button
          v-if="state.id"
          class="btn btn-outline-danger"
          @click="promptDelete">
          {{ t('buttons.delete') }}
        </button>
      </div>
    </template>
  </ModalWindow>

  <ModalWindow
    v-if="askDelete"
    class="confirm-modal"
    :close-modal="() => (askDelete = false)">
    <template #header>
      <h3>{{ t('message.deleteCategory') }}?</h3>
    </template>
    <template #main>
      <p>{{ t('message.areYouSureWantDeleteCategory') }}</p>
      <div class="actions center">
        <button
          class="btn btn-outline-primary"
          @click="askDelete = false">
          {{ t('buttons.cancel') }}
        </button>
        <button
          class="btn btn-danger"
          @click="reallyDelete">
          {{ t('buttons.delete') }}
        </button>
      </div>
    </template>
  </ModalWindow>
</template>

<style scoped>
.form-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  margin-bottom: 24px;
}
.form-input {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid #ced4da;
  border-radius: 4px;
}

.chk {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-row {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}
.btn.danger {
  color: #dc3545;
  border-color: #dc3545;
}
.btn.danger:hover {
  background: #dc3545;
  color: #fff;
}
.ed-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}
.ed-header .title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.ed-header i.income {
  color: #22c55e;
}
.ed-header i.expense {
  color: #ef4444;
}
.pill {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}
.pill.income {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
}
.pill.expense {
  background: rgba(239, 68, 68, 0.15);
  color: #dc2626;
}
.muted {
  color: #6c757d;
  font-size: 12px;
}

.ed-grid {
  display: grid;
  gap: 24px;
  grid-template-columns: minmax(260px, 1fr) minmax(220px, 0.8fr);
  margin-top: 8px;
}
.col.side {
  align-self: start;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.label {
  font-size: 13px;
  color: #6c757d;
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid #ced4da;
  border-radius: 4px;
}

.segmented {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.segmented .btn.outline.active {
  background: #1e90ff;
  color: #fff;
  border-color: #1e90ff;
}

.switch {
  flex-direction: row;
  align-items: center;
  gap: 8px;
}

.hint-card {
  background: #f7fbff;
  border: 1px solid #e5f0ff;
  border-radius: 8px;
  padding: 12px;
  font-size: 13px;
  color: #445;
}
.hint-title {
  margin: 0 0 6px;
  font-weight: 600;
  font-size: 14px;
}
.hint-card ul {
  margin: 0;
  padding-left: 18px;
}

.actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 16px;
}
.actions.center {
  justify-content: center;
}
.btn.danger {
  color: #dc3545;
  border-color: #dc3545;
}
.btn.danger:hover {
  background: #dc3545;
  color: #fff;
}

:deep(.modal-content) {
  max-width: 680px;
}

:deep(.confirm-modal .modal-content) {
  max-width: 360px;
}

@media (max-width: 768px) {
  .ed-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 576px) {
  .ed-header .title {
    gap: 8px;
  }
  :deep(.modal-content) {
    width: 95%;
  }
}
</style>
