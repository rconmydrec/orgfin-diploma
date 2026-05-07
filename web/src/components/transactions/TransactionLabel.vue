<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue';
import { useUserStore } from '@/stores/user';
import debounce from 'lodash/debounce';

const props = defineProps({
  transaction: {
    type: Object,
    required: true,
  },
  isTransfer: {
    type: Boolean,
    default: false,
  },
});
const emit = defineEmits(['update:label']);

const userStore = useUserStore();
const label = ref(props.transaction.label || '');
const inputEl = ref(null);
const suggestions = ref([]);
const insideClick = ref(false);
const activeIndex = ref(-1);

const filterSuggestions = debounce(() => {
  if (!label.value.trim() || label.value.length < 3) {
    suggestions.value = [];
    activeIndex.value = -1;
    return;
  }
  const searchTerm = label.value.toLowerCase();
  suggestions.value = userStore.transactionTemplates.filter((t) => {
    return t.label.toLowerCase().includes(searchTerm);
  });
  activeIndex.value = 0;
}, 500);

function selectSuggestion(selectedTemplate) {
  label.value = selectedTemplate.label;
  emit('update:label', selectedTemplate);
  suggestions.value = [];
  activeIndex.value = -1;
}

watch(
  () => props.transaction.label,
  (newLabel) => {
    label.value = newLabel;
  },
);

function labelChanged($event) {
  label.value = $event.target.value;
  emit('update:label', label.value);
  filterSuggestions();
}

function handleKeyDown(event) {
  if (suggestions.value.length === 0) return;

  if (event.key === 'ArrowDown') {
    event.preventDefault();
    activeIndex.value = (activeIndex.value + 1) % suggestions.value.length;
  } else if (event.key === 'ArrowUp') {
    event.preventDefault();
    activeIndex.value =
      (activeIndex.value - 1 + suggestions.value.length) %
      suggestions.value.length;
  } else if (event.key === 'Enter') {
    event.preventDefault();
    if (
      activeIndex.value >= 0 &&
      activeIndex.value < suggestions.value.length
    ) {
      selectSuggestion(suggestions.value[activeIndex.value]);
    }
  } else if (event.key === 'Escape') {
    suggestions.value = [];
    activeIndex.value = -1;
  }
}

function blurHandler() {
  setTimeout(() => {
    if (!insideClick.value) {
      suggestions.value = [];
      activeIndex.value = -1;
    }
  }, 10);
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown);

  inputEl.value?.focus();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown);
});
</script>

<template>
  <div class="mb-3 position-relative">
    <label
      for="label"
      class="form-label">
      {{ $t('message.label') }}
    </label>
    <input
      type="text"
      class="form-control"
      @input="labelChanged"
      id="label"
      :value="label"
      autocomplete="off"
      ref="inputEl"
      @focus="filterSuggestions"
      @blur="blurHandler" />

    <ul
      v-if="suggestions.length"
      @mousedown="insideClick = true"
      @mouseup="insideClick = false"
      class="list-group position-absolute w-100 mt-1 shadow">
      <li
        v-for="(suggestion, index) in suggestions"
        :key="suggestion.id"
        class="list-group-item list-group-item-action"
        :class="{ active: index === activeIndex }"
        @click="() => selectSuggestion(suggestion)">
        {{ suggestion.label }}
        <b v-if="suggestion.targetAccountId">
          (-> {{ suggestion.targetAccountName }})
        </b>
        <b v-else-if="suggestion.category">({{ suggestion.category.name }})</b>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.position-relative {
  position: relative;
}

.list-group {
  z-index: 10;
  max-height: 200px;
  overflow-y: auto;
}

.list-group-item.active {
  background-color: #007bff;
  color: white;
}
</style>
