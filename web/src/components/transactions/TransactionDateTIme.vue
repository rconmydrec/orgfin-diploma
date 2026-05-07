<script setup>
import { DateTime } from 'luxon';
import { ref, watchEffect } from 'vue';

const props = defineProps({ isEdit: Boolean, transaction: Object });
const emit = defineEmits(['dateTimeChanged']);

const currentDate = DateTime.now().toISODate(); // 'YYYY-MM-DD'
const currentTime = DateTime.now().toFormat('HH:mm'); // 'HH:MM'

const date = ref(currentDate);
const time = ref(currentTime);

watchEffect(() => {
  if (props.isEdit && props.transaction && props.transaction.dateTime) {
    const transactionDateTime = DateTime.fromISO(props.transaction.dateTime);
    date.value = transactionDateTime.toISODate();
    time.value = transactionDateTime.toFormat('HH:mm');
  }
});

function dateOrTimeChanged() {
  emit('dateTimeChanged', { date: date.value, time: time.value });
}
</script>

<template>
  <div class="row">
    <div class="col-6">
      <div class="form-group">
        <label for="transaction-date">Date</label>
        <input
          type="date"
          class="form-control"
          id="transaction-date"
          v-model="date"
          @change="dateOrTimeChanged" />
      </div>
    </div>
    <div class="col-6">
      <div class="form-group">
        <label for="transaction-time">Time</label>
        <input
          type="time"
          class="form-control"
          id="transaction-time"
          v-model="time"
          @change="dateOrTimeChanged" />
      </div>
    </div>
  </div>
</template>

<style scoped></style>
