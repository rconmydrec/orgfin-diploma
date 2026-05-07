<script setup>
import { ref, onBeforeMount, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Services } from '@/services/servicesConfig';
import { useUserStore } from '@/stores/user';
import EditTemplateModal from './EditTemplateModal.vue';

const userStore = useUserStore();
const { t } = useI18n();

const selectedTemplates = ref([]);
const filtered = ref([]);
const query = ref('');
const typeFilter = ref('all');
const editingTemplate = ref(null);

/**
 * Derive the transaction type of a template from its data fields.
 * @param {object} tpl - template object
 * @returns {'transfer'|'income'|'expense'} derived type
 */
function getTemplateType(tpl) {
  if (tpl.targetAccountId != null) return 'transfer';
  if (tpl.category?.isIncome) return 'income';
  return 'expense';
}

/**
 * Apply both search query and type filter to the full template list.
 */
function applyFilters() {
  filtered.value = userStore.transactionTemplates.filter((tpl) => {
    const matchesQuery = tpl.label
      .toLowerCase()
      .includes(query.value.toLowerCase());
    const matchesType =
      typeFilter.value === 'all' || getTemplateType(tpl) === typeFilter.value;
    return matchesQuery && matchesType;
  });
}

onBeforeMount(async () => {
  const data = await Services.transactionsService.getUserTemplates();
  userStore.transactionTemplates.splice(
    0,
    userStore.transactionTemplates.length,
    ...data,
  );
  applyFilters();
});

watch([query, typeFilter], () => {
  applyFilters();
});

function openEditModal(template) {
  editingTemplate.value = template;
}

function closeEditModal() {
  editingTemplate.value = null;
}

async function saveTemplate(templateData) {
  try {
    await Services.transactionsService.updateUserTemplate(templateData);
    const upd = await Services.transactionsService.getUserTemplates();
    userStore.transactionTemplates = upd;
    applyFilters();
    closeEditModal();
  } catch (error) {
    console.error('Error updating template:', error);
  }
}

async function remove() {
  await Services.transactionsService.deleteUserTemplates(
    selectedTemplates.value,
  );
  const upd = await Services.transactionsService.getUserTemplates();
  userStore.transactionTemplates = upd;
  selectedTemplates.value = [];
  applyFilters();
}
</script>

<template>
  <div class="section-card templates-wrapper">
    <h3>{{ t('message.templates') }}</h3>

    <input
      v-model="query"
      class="form-control search"
      :placeholder="t('message.searchTemplates')" />

    <ul class="nav nav-pills type-filter">
      <li class="nav-item">
        <a
          class="nav-link"
          :class="{ active: typeFilter === 'all' }"
          href="#"
          @click.prevent="typeFilter = 'all'">
          {{ t('message.all') }}
        </a>
      </li>
      <li class="nav-item">
        <a
          class="nav-link"
          :class="{ active: typeFilter === 'expense' }"
          href="#"
          @click.prevent="typeFilter = 'expense'">
          {{ t('message.expense') }}
        </a>
      </li>
      <li class="nav-item">
        <a
          class="nav-link"
          :class="{ active: typeFilter === 'income' }"
          href="#"
          @click.prevent="typeFilter = 'income'">
          {{ t('message.income') }}
        </a>
      </li>
      <li class="nav-item">
        <a
          class="nav-link"
          :class="{ active: typeFilter === 'transfer' }"
          href="#"
          @click.prevent="typeFilter = 'transfer'">
          {{ t('message.transfer') }}
        </a>
      </li>
    </ul>

    <div class="tmpl-list">
      <div
        v-for="tpl in filtered"
        :key="tpl.id"
        class="tmpl-item">
        <label class="tmpl-item-label">
          <input
            type="checkbox"
            :value="tpl.id"
            v-model="selectedTemplates" />
          <span>
            {{ tpl.label }}
            <b v-if="tpl.targetAccountId">(-> {{ tpl.targetAccountName }})</b>
            <b v-else-if="tpl.category">({{ tpl.category.name }})</b>
          </span>
        </label>
        <button
          type="button"
          class="btn btn-sm btn-outline-primary edit-btn"
          @click="openEditModal(tpl)">
          <i class="fa fa-pencil"></i>
        </button>
      </div>
    </div>

    <button
      class="btn btn-primary"
      :disabled="!selectedTemplates.length"
      @click="remove">
      {{ t('buttons.delete') }}
    </button>

    <EditTemplateModal
      :template="editingTemplate"
      @close="closeEditModal"
      @save="saveTemplate" />
  </div>
</template>

<style scoped>
.search {
  margin: 0 0 16px;
}
.type-filter {
  margin-bottom: 16px;
}
.tmpl-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 300px;
  overflow-y: auto;
  margin-bottom: 16px;
}
.tmpl-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background-color 0.2s;
}
.tmpl-item:hover {
  background-color: #f8f9fa;
}
.tmpl-item-label {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  margin: 0;
}
.edit-btn {
  padding: 2px 8px;
  font-size: 0.875rem;
}
</style>
