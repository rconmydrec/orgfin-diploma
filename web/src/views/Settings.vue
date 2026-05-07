<script setup>
import { ref, shallowRef } from 'vue';

import UserProfile from '@/components/settings/UserProfile.vue';
import CategoriesManager from '@/components/settings/CategoriesManager.vue';
import TransactionTemplates from '@/components/templates/TransactionTemplates.vue';
import ExportTransactions from '@/components/settings/ExportTransactions.vue';

const sections = [
  {
    id: 'profile',
    labelKey: 'buttons.profile',
    icon: 'fa-user-circle',
    comp: UserProfile,
  },
  {
    id: 'templates',
    labelKey: 'buttons.templates',
    icon: 'fa-copy',
    comp: TransactionTemplates,
  },
  {
    id: 'categories',
    labelKey: 'buttons.categories',
    icon: 'fa-layer-group',
    comp: CategoriesManager,
  },
  {
    id: 'export',
    labelKey: 'buttons.export',
    icon: 'fa-file-export',
    comp: ExportTransactions,
  },
];

const activeId = ref('profile');
const ActiveComp = shallowRef(sections[0].comp);

function changeSection(id) {
  if (id === activeId.value) return;
  const found = sections.find((s) => s.id === id);
  if (found) {
    activeId.value = id;
    ActiveComp.value = found.comp;
  }
}
</script>

<template>
  <main class="settings-page">
    <div class="container split">
      <aside class="sidebar">
        <h2 class="title">{{ $t('message.settings') }}</h2>

        <nav class="side-nav">
          <button
            v-for="s in sections"
            :key="s.id"
            class="btn side-btn"
            :class="{ active: s.id === activeId }"
            @click="changeSection(s.id)">
            <i :class="['fa-solid', s.icon]"></i>
            <span>{{ $t(s.labelKey) }}</span>
          </button>
        </nav>
      </aside>

      <section class="settings-content panel">
        <component :is="ActiveComp" />
      </section>
    </div>
  </main>
</template>
