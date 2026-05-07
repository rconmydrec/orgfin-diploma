<script setup>
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';

const emit = defineEmits(['change']);

const { locale } = useI18n();
const showLangDropdown = ref(false);
const languages = ref([]);

onMounted(async () => {
  try {
    const BACKEND_HOST = import.meta.env.VITE_BACKEND_HOST;
    const response = await fetch(`${BACKEND_HOST}/settings/languages`);
    if (response.ok) {
      languages.value = await response.json();
    }
  } catch (err) {
    console.error('Failed to load languages:', err);
  }
});

function changeLanguage(langCode) {
  locale.value = langCode;
  showLangDropdown.value = false;
  emit('change', langCode);
}

function toggleLangDropdown() {
  showLangDropdown.value = !showLangDropdown.value;
}

const currentLanguage = () => {
  return (
    languages.value.find((l) => l.code === locale.value) ||
    languages.value[0] || { code: 'en', name: 'EN' }
  );
};

const displayCode = (code) => {
  if (code === 'uk') return 'UA';
  if (code === 'zh') return 'CN';
  return code.toUpperCase();
};
</script>

<template>
  <div class="language-switcher">
    <button
      type="button"
      class="lang-dropdown-toggle"
      @click="toggleLangDropdown">
      <i class="fa fa-globe"></i>
      {{ displayCode(currentLanguage().code) }}
      <i class="fa fa-chevron-down"></i>
    </button>
    <div
      v-if="showLangDropdown"
      class="lang-dropdown">
      <button
        v-for="lang in languages"
        :key="lang.id"
        type="button"
        @click="changeLanguage(lang.code)"
        :class="{ active: locale === lang.code }">
        {{ lang.name }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.language-switcher {
  position: relative;
}

.lang-dropdown-toggle {
  padding: 6px 12px;
  background: white;
  border: 1px solid #ddd;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  gap: 6px;
  color: #333;
  font-size: 14px;
}

.lang-dropdown-toggle:hover {
  background: #f5f5f5;
  border-color: #bbb;
}

.lang-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 4px;
  background: white;
  border: 1px solid #ddd;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  min-width: 150px;
  z-index: 1000;
}

.lang-dropdown button {
  width: 100%;
  padding: 10px 16px;
  background: white;
  border: none;
  cursor: pointer;
  text-align: left;
  transition: background 0.2s;
  color: #333;
  font-weight: 400;
}

.lang-dropdown button:hover {
  background: #f5f5f5;
}

.lang-dropdown button.active {
  background: #007bff;
  color: white;
  font-weight: 500;
}

@media (max-width: 576px) {
  .lang-dropdown-toggle {
    padding: 5px 10px;
    gap: 5px;
    font-size: 13px;
  }
}

@media (max-width: 440px) {
  .lang-dropdown-toggle .fa-globe {
    display: none;
  }

  .lang-dropdown-toggle {
    padding: 5px 8px;
    gap: 4px;
    font-size: 12px;
  }
}
</style>
