<script setup>
const props = defineProps(['transaction', 'categories']);
const emit = defineEmits(['update:categoryId']);

function changeCategory($event) {
  emit('update:categoryId', $event.target.value);
}
</script>

<template>
  <label
    for="category-select"
    class="form-label">
    {{ $t('message.category') }}
  </label>
  <select
    id="category-select"
    class="form-select bottom-space"
    @change="changeCategory"
    :value="props.transaction.categoryId">
    <option
      v-for="category in props.categories"
      :key="category.id"
      :value="category.id">
      {{ category.name }}
    </option>
  </select>
</template>
