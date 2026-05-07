<script setup>
import { ref, watch } from 'vue';

const emit = defineEmits(['update-dates']);

const startDate = ref('');
const endDate = ref('');

const emitDates = () => {
  emit('update-dates', { startDate: startDate.value, endDate: endDate.value });
};

watch([startDate, endDate], emitDates);
</script>

<template>
  <div class="row g-3">
    <div class="col-md">
      <label
        for="startDate"
        class="form-label">
        From
      </label>
      <div class="d-flex">
        <input
          type="date"
          class="form-control"
          id="startDate"
          v-model="startDate"
          placeholder="From" />
        <button
          type="button"
          class="btn-clear btn-close"
          @click="startDate = ''"
          aria-label="Clear"></button>
      </div>
    </div>
    <div class="col-md">
      <label
        for="endDate"
        class="form-label">
        To
      </label>
      <div class="d-flex">
        <input
          type="date"
          class="form-control"
          id="endDate"
          v-model="endDate"
          placeholder="To" />
        <button
          type="button"
          class="btn-clear btn-close"
          @click="endDate = ''"
          aria-label="Clear"></button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.d-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
