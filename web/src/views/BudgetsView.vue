<script setup>
import { onBeforeMount, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';
import BudgetsList from '@/components/budgets/BudgetsList.vue';
import BudgetNewComponent from '@/components/budgets/BudgetNewComponent.vue';

const router = useRouter();

let activeBudgets = reactive([]);
let archivedBudgets = reactive([]);
let currentBudget = reactive({});
const showArchived = ref(false);

onBeforeMount(async () => {
  await fetchBudgets();
});

const fetchBudgets = async () => {
  try {
    const response = await Services.budgetsService.getUserBudgets('active');
    activeBudgets.splice(0);
    activeBudgets.push(...response);
  } catch (e) {
    await processError(e, router);
  }
};

const showBudgetModal = ref(false);

const openBudgetModal = () => {
  showBudgetModal.value = true;
};

const startCreateBudget = () => {
  currentBudget = {};
  openBudgetModal();
};

const closeBudgetModal = () => {
  showBudgetModal.value = false;
};

const budgetCreated = (budget) => {
  currentBudget = budget;
  fetchBudgets();
};

const editBudget = (budget) => {
  currentBudget = budget;
  openBudgetModal();
};

const showArchivedBudgets = async () => {
  try {
    const response = await Services.budgetsService.getUserBudgets('archived');
    archivedBudgets.splice(0);
    archivedBudgets.push(...response);
  } catch (e) {
    await processError(e, router);
  }
  showArchived.value = !showArchived.value;
};

const hideArchivedBudgets = () => {
  showArchived.value = false;
};
</script>

<template>
  <main>
    <div class="container">
      <div class="row">
        <div class="col-4">
          <button
            class="btn btn-primary w-100"
            @click="startCreateBudget">
            {{ $t('buttons.newBudget') }}
          </button>
        </div>
      </div>

      <div class="row">
        <BudgetsList
          :budgets="activeBudgets"
          @budget-selected="editBudget" />
      </div>

      <div class="row">
        <div class="col-4 mt-4">
          <button
            v-if="!showArchived"
            class="btn btn-primary w-100"
            @click="showArchivedBudgets">
            <span>{{ $t('buttons.showArchivedBudgets') }}</span>
          </button>

          <button
            v-if="showArchived"
            class="btn btn-primary w-100"
            @click="hideArchivedBudgets">
            <span>{{ $t('buttons.hideArchivedBudgets') }}</span>
          </button>
        </div>
      </div>

      <div
        v-if="showArchived"
        class="row">
        <BudgetsList
          :budgets="archivedBudgets"
          @budget-selected="editBudget" />
      </div>

      <teleport to="body">
        <BudgetNewComponent
          v-if="showBudgetModal"
          @budget-created="budgetCreated"
          @budget-deleted="fetchBudgets"
          @budget-archived="fetchBudgets"
          :edit-budget="currentBudget"
          :close-modal="closeBudgetModal" />
      </teleport>
    </div>
  </main>
</template>

<style scoped lang="scss"></style>
