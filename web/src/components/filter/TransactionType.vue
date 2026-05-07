<script setup>
const props = defineProps(['types']);
const emit = defineEmits(['transactionTypeChanged']);

const types = { ...props.types };

function transactionTypeChanged(event) {
  const status = event.target.checked;
  const transType = event.target.value;
  types[transType] = status;
  emit('transactionTypeChanged', {
    newTransactionTypes: types,
  });
}
</script>

<template>
  <div class="row">
    <div class="col">
      <div class="checkboxes-container">
        <span class="one-checkbox-container form-check form-switch">
          <input
            class="form-check-input"
            type="checkbox"
            value="expense"
            id="expense-check"
            @click="transactionTypeChanged"
            :checked="props.types.expense" />
          <label
            class="form-check-label"
            for="expense-check">
            Expense
          </label>
        </span>
        <span class="one-checkbox-container form-check form-switch">
          <input
            class="form-check-input"
            type="checkbox"
            value="income"
            id="income-check"
            @click="transactionTypeChanged"
            :checked="props.types.income" />
          <label
            class="form-check-label"
            for="income-check">
            Income
          </label>
        </span>
        <span class="one-checkbox-container form-check form-switch">
          <input
            class="form-check-input"
            type="checkbox"
            value="transfer"
            id="transfer-check"
            @click="transactionTypeChanged"
            :checked="props.types.transfer" />
          <label
            class="form-check-label"
            for="transfer-check">
            Transfer
          </label>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.checkboxes-container {
  display: flex;
  justify-content: start;
}
.one-checkbox-container {
  margin-right: 5vw;
}
.form-check-input {
  margin-right: 1vw;
}
</style>
