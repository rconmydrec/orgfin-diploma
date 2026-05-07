<template>
  <div
    v-if="modelValue"
    class="modal fade show"
    style="display: block"
    tabindex="-1"
    @click.self="cancel">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">
            <i class="fa-solid fa-triangle-exclamation text-warning me-2"></i>
            {{ title }}
          </h5>
          <button
            type="button"
            class="btn-close"
            @click="cancel"></button>
        </div>
        <div class="modal-body">
          {{ message }}
        </div>
        <div class="modal-footer">
          <button
            v-if="showCancel"
            type="button"
            class="btn btn-secondary"
            @click="cancel">
            {{ cancelText }}
          </button>
          <button
            type="button"
            class="btn btn-danger"
            @click="confirm">
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  </div>
  <div
    v-if="modelValue"
    class="modal-backdrop fade show"></div>
</template>

<script setup>
const props = defineProps({
  modelValue: {
    type: Boolean,
    required: true,
  },
  title: {
    type: String,
    default: 'Confirm',
  },
  message: {
    type: String,
    required: true,
  },
  confirmText: {
    type: String,
    default: 'Delete',
  },
  cancelText: {
    type: String,
    default: 'Cancel',
  },
  showCancel: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel']);

function confirm() {
  emit('confirm');
  emit('update:modelValue', false);
}

function cancel() {
  emit('cancel');
  emit('update:modelValue', false);
}
</script>

<style scoped>
.modal {
  z-index: 1055;
}

.modal-backdrop {
  z-index: 1050;
}
</style>
