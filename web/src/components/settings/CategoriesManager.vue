<script setup>
import { onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { Services } from '@/services/servicesConfig';
import CategoryEditor from './CategoryEditor.vue';

const { t } = useI18n();

const cats = reactive({ income: [], expenses: [] });
const showEditor = ref(false);
const editableCat = ref({});
const catPool = ref([]);

async function loadCats() {
  const grouped = await Services.categoriesService.getGroupedCategories();
  cats.income = [...grouped.income].sort((a, b) =>
    a.name.localeCompare(b.name),
  );
  cats.expenses = [...grouped.expenses].sort((a, b) =>
    a.name.localeCompare(b.name),
  );
}
onMounted(loadCats);

function openEditor(cat = { children: [] }) {
  editableCat.value = cat;
  catPool.value = cat.isIncome ? cats.income : cats.expenses;
  showEditor.value = true;
}
function closeEditor() {
  showEditor.value = false;
}
async function updated() {
  await loadCats();
}
</script>

<template>
  <div class="section-card cat-card">
    <div class="cat-header">
      <div class="title-wrap">
        <h3>{{ t('message.categories') }}</h3>
        <div class="stats">
          <span class="stat">
            {{ t('message.income') }}: {{ cats.income.length }}
          </span>
          <span class="dot">•</span>
          <span class="stat">
            {{ t('message.expense') }}: {{ cats.expenses.length }}
          </span>
        </div>
      </div>

      <button
        class="btn btn-primary"
        @click="openEditor()">
        <i class="fa-solid fa-plus"></i>
        {{ t('buttons.addNewCategory') }}
      </button>
    </div>

    <div class="cat-groups">
      <details
        class="cat-accord"
        open>
        <summary>
          <span>{{ t('message.income') }}</span>
          <span class="badge">{{ cats.income.length }}</span>
        </summary>

        <div class="cat-grid">
          <div
            v-for="c in cats.income"
            :key="c.id"
            class="cat-tile">
            <button
              class="btn cat-btn"
              @click="openEditor(c)">
              <i class="fa-solid fa-circle-up"></i>
              <span class="name">{{ c.name }}</span>
            </button>

            <div
              v-if="c.children.length"
              class="chips">
              <button
                v-for="sc in c.children"
                :key="sc.id"
                class="btn cat-chip"
                @click="openEditor(sc)">
                {{ sc.name }}
              </button>
            </div>
          </div>
        </div>
      </details>

      <details
        class="cat-accord"
        open>
        <summary>
          <span>{{ t('message.expense') }}</span>
          <span class="badge">{{ cats.expenses.length }}</span>
        </summary>

        <div class="cat-grid">
          <div
            v-for="c in cats.expenses"
            :key="c.id"
            class="cat-tile">
            <button
              class="btn cat-btn"
              @click="openEditor(c)">
              <i class="fa-solid fa-circle-down"></i>
              <span class="name">{{ c.name }}</span>
            </button>

            <div
              v-if="c.children.length"
              class="chips">
              <button
                v-for="sc in c.children"
                :key="sc.id"
                class="btn cat-chip"
                @click="openEditor(sc)">
                {{ sc.name }}
              </button>
            </div>
          </div>
        </div>
      </details>
    </div>
  </div>

  <CategoryEditor
    v-if="showEditor"
    :category="editableCat"
    :categories="catPool"
    :all-categories="cats"
    @category-updated="updated"
    :close-modal="closeEditor" />
</template>

<style scoped>
.cat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.title-wrap {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}
.title-wrap h3 {
  margin: 0;
}
.stats {
  display: flex;
  gap: 10px;
  color: #6c757d;
  font-size: 14px;
}
.dot {
  opacity: 0.6;
}

.cat-accord {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 8px 12px 12px;
}
.cat-accord + .cat-accord {
  margin-top: 16px;
}

.cat-accord > summary {
  list-style: none;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
  font-weight: 600;
  margin: 4px 0 10px;
}
.cat-accord > summary::-webkit-details-marker {
  display: none;
}

.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  background: rgba(30, 144, 255, 0.12);
  color: #1e90ff;
}

.cat-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.cat-tile {
  background: #f9fbff;
  border: 1px solid #e7eef9;
  border-radius: 8px;
  padding: 10px;
  transition:
    box-shadow 0.15s,
    transform 0.15s;
}
.cat-tile:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}

.cat-btn {
  width: 100%;
  justify-content: flex-start;
  gap: 10px;
  padding: 10px 12px;
  background: #eaf4ff;
  color: #1e90ff;
  border: none;
  border-radius: 6px;
  font-weight: 600;
  transition: background 0.15s;
}
.cat-btn i {
  font-size: 14px;
}
.cat-btn:hover {
  background: #d9ebff;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.cat-chip {
  padding: 6px 10px;
  background: #fff;
  border: 1px solid #dbe7ff;
  color: #1e90ff;
  border-radius: 999px;
  font-size: 13px;
  transition:
    background 0.15s,
    border-color 0.15s;
}
.cat-chip:hover {
  background: #f2f7ff;
  border-color: #cfe0ff;
}

@media (max-width: 576px) {
  .cat-btn {
    padding: 8px 10px;
  }
  .cat-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .cat-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }
  .cat-header .btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
