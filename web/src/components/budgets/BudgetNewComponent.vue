<script setup>
import { onBeforeMount, reactive, ref, watch } from 'vue';

import ModalWindow from '@/components/utils/ModalWindow.vue';
import CategoriesFilter from '@/components/filter/CategoriesFilter.vue';
import CalculatorInput from '@/components/utilities/CalculatorInput.vue';
import { processError } from '@/errors/errorHandlers';
import { useCategoriesStore } from '@/stores/categories';
import { Services } from '@/services/servicesConfig';
import { RouterLink, useRouter } from 'vue-router';

const props = defineProps({
  closeModal: Function,
  editBudget: Object,
});

const router = useRouter();
const isEdit = !!props.editBudget?.id;

const emit = defineEmits([
  'budgetCreated',
  'budgetUpdated',
  'budgetDeleted',
  'budgetArchived',
]);

let userCategories = reactive([]);

const budgetName = ref(props.editBudget?.name ?? '');
let currency = reactive(props.editBudget?.currency ?? {});
const targetAmount = ref(props.editBudget?.targetAmount ?? 0);
const period = ref(props.editBudget?.period ?? '');
const repeat = ref(props.editBudget?.repeat ?? false);
const startDate = ref(props.editBudget?.startDate?.split('T')[0] ?? '');
const endDate = ref(props.editBudget?.endDate?.split('T')[0] ?? '');
const comment = ref(props.editBudget?.comment ?? '');
const currencies = reactive([]);

let categories = reactive([]);
const showCategoriesModal = ref(false);
const showDeleteModal = ref(false);
const showArchiveModal = ref(false);

onBeforeMount(async () => {
  userCategories = useCategoriesStore().categories;
  if (userCategories.length === 0) {
    try {
      await Services.categoriesService.getUserCategories();
    } catch (e) {
      await processError(e, router);
    }
    userCategories = useCategoriesStore().categories;
  }
  currencies.push(...(await Services.currenciesService.getAllCurrencies()));
  getCategoriesFromUserCategories(props.editBudget?.includedCategories ?? '');
});

function getCategoriesFromUserCategories(categoriesIds) {
  const budgetCategories = categoriesIds
    .split(',')
    .map((id) => parseInt(id, 10));
  const cats = userCategories.filter((category) => {
    return budgetCategories.includes(category.id);
  });
  categories.length = 0;
  categories.push(...cats);
}

watch(currencies, (newCurrencies) => {
  if (newCurrencies.length > 0 && props.editBudget.currency) {
    const matchedCurrency = newCurrencies.find(
      (c) => c.id === props.editBudget.currency.id,
    );
    if (matchedCurrency) {
      currency = matchedCurrency;
    }
  }
});

function openCategoriesModal() {
  showCategoriesModal.value = true;
}

function closeCategoriesModal() {
  showCategoriesModal.value = false;
}

function categoriesUpdated(selectedCategories) {
  categories.length = 0;
  userCategories.forEach((category) => {
    if (selectedCategories.includes(category.id)) {
      categories.push(category);
    }
  });
  showCategoriesModal.value = false;
}

function extractCategoriesIds(categories) {
  return categories.map((category) => category.id);
}

async function submitForm() {
  const budgetObj = {
    name: budgetName.value,
    currencyId: currency.id,
    targetAmount: targetAmount.value,
    period: period.value,
    repeat: repeat.value,
    startDate: startDate.value,
    endDate: endDate.value,
    comment: comment.value,
    categories: extractCategoriesIds(categories),
  };

  let createdBudget;
  if (isEdit) {
    budgetObj.id = props.editBudget.id;
    createdBudget = await Services.budgetsService.updateBudget(budgetObj);
  } else {
    createdBudget = await Services.budgetsService.createBudget(budgetObj);
  }

  if (createdBudget) {
    emit('budgetCreated', createdBudget);
  }

  props.closeModal();
}

async function deleteBudget() {
  if (isEdit) {
    await Services.budgetsService.deleteBudget(props.editBudget.id);
    emit('budgetDeleted', props.editBudget.id);
    showDeleteModal.value = false;
    props.closeModal();
  }
}

async function confirmDeleteBudget() {
  showDeleteModal.value = true;
}

async function archiveBudget() {
  if (isEdit) {
    await Services.budgetsService.archiveBudget(props.editBudget.id);
    emit('budgetArchived', props.editBudget.id);
    showArchiveModal.value = false;
    props.closeModal();
  }
}

async function confirmArchiveBudget() {
  showArchiveModal.value = true;
}
</script>

<template>
  <ModalWindow :close-modal="closeModal">
    <template #header>
      <div class="row">
        <div class="header-container">
          <h2>{{ $t('message.budgetDetails') }}</h2>
          <span>
            <RouterLink
              :to="{
                name: 'transactions',
                query: {
                  categories: categories
                    .map((category) => category.id)
                    .join(','),
                  startDate: startDate,
                  endDate: endDate,
                },
              }">
              <button class="btn btn-secondary">
                {{ $t('menu.transactions') }}
              </button>
            </RouterLink>
          </span>
        </div>
      </div>
    </template>

    <template #main>
      <form @submit.prevent="submitForm">
        <div class="mb-3">
          <label
            for="name"
            class="form-label">
            Name
          </label>
          <input
            type="text"
            class="form-control"
            id="name"
            v-model="budgetName"
            maxlength="100"
            required />
        </div>

        <div class="mb-3">
          <label
            for="currencyId"
            class="form-label">
            Select Currency
          </label>
          <select
            id="currency_id"
            class="form-control"
            v-model="currency"
            required>
            <option
              v-for="currency in currencies"
              :key="currency.id"
              :value="currency">
              {{ currency.name }}
            </option>
          </select>
        </div>

        <div class="mb-3">
          <label
            for="target_amount"
            class="form-label">
            Target Amount
          </label>
          <CalculatorInput
            id="target_amount"
            v-model="targetAmount"
            :placeholder="'0.00'"
            :min="0" />
        </div>

        <div class="mb-3">
          <label
            for="period"
            class="form-label">
            Period
          </label>
          <select
            class="form-select"
            id="period"
            v-model="period"
            required>
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
            <option value="yearly">Yearly</option>
            <option value="custom">Custom</option>
          </select>
        </div>

        <div class="form-check form-switch mb-3">
          <input
            type="checkbox"
            class="form-check-input"
            id="repeat"
            v-model="repeat" />
          <label
            class="form-check-label"
            for="repeat">
            Repeat
          </label>
        </div>

        <div class="mb-3">
          <label
            for="start_date"
            class="form-label">
            Start Date
          </label>
          <input
            type="date"
            class="form-control"
            id="start_date"
            v-model="startDate"
            required />
        </div>

        <div class="mb-3">
          <label
            for="end_date"
            class="form-label">
            End Date
          </label>
          <input
            type="date"
            class="form-control"
            id="end_date"
            v-model="endDate"
            required />
        </div>

        <div class="mb-3">
          <label
            for="categories"
            class="form-label">
            {{ $t('message.categories') }}
          </label>
          <div
            v-for="category in categories"
            :key="category.id">
            {{ category.name }}
          </div>
        </div>

        <div class="mb-3">
          <a
            href="#"
            class="btn btn-primary"
            @click.stop="openCategoriesModal">
            {{ $t('message.selectCategories') }}
          </a>
          <teleport to="body">
            <CategoriesFilter
              v-if="showCategoriesModal"
              :close-modal="closeCategoriesModal"
              :initial-categories="extractCategoriesIds(categories)"
              @categories-updated="categoriesUpdated" />
          </teleport>
        </div>

        <div class="mb-3">
          <label
            for="comment"
            class="form-label">
            Comment
          </label>
          <textarea
            class="form-control"
            id="comment"
            v-model="comment"
            rows="3"></textarea>
        </div>

        <div class="manage-buttons">
          <button
            type="submit"
            class="btn btn-primary">
            Submit
          </button>
          <button
            type="button"
            class="btn btn-warning"
            @click.prevent="confirmArchiveBudget">
            {{ $t('buttons.archive') }}
          </button>
          <button
            type="button"
            class="btn btn-danger"
            @click="confirmDeleteBudget">
            {{ $t('buttons.delete') }}
          </button>
        </div>
      </form>

      <ModalWindow
        :close-modal="() => (showDeleteModal = false)"
        v-if="showDeleteModal">
        <template #header>
          <h2>{{ $t('message.deleteBudget') }}?</h2>
        </template>

        <template #main>
          <p>{{ $t('message.areYouSureWantDeleteBudget') }}?</p>
          <div class="row gap-2 justify-content-center">
            <button
              class="btn btn-danger col-auto"
              @click="deleteBudget">
              {{ $t('buttons.delete') }}
            </button>
          </div>
        </template>
      </ModalWindow>

      <ModalWindow
        :close-modal="() => (showArchiveModal = false)"
        v-if="showArchiveModal">
        <template #header>
          <h2>{{ $t('message.archiveBudget') }}</h2>
        </template>

        <template #main>
          <p>{{ $t('message.areYouSureWantArchiveBudget') }}?</p>
          <div class="row gap-2 justify-content-center">
            <button
              class="btn btn-danger col-auto"
              @click="archiveBudget">
              {{ $t('buttons.archive') }}
            </button>
          </div>
        </template>
      </ModalWindow>
    </template>
  </ModalWindow>
</template>

<style scoped>
.header-container {
  display: flex;
  justify-content: space-between;
}

.form-label {
  font-weight: bold;
}

.manage-buttons {
  display: flex;
  justify-content: space-between;
}
</style>
