<script setup>
const props = defineProps({
  closeModal: Function,
  showCloseButton: {
    type: Boolean,
    default: true,
  },
});

function closeButtonClicked() {
  props.closeModal();
}
</script>

<template>
  <div
    class="modal-overlay"
    @click="closeButtonClicked">
    <div
      class="modal-content"
      @click.stop>
      <slot name="header"></slot>

      <div class="container">
        <slot name="main"></slot>
      </div>

      <div
        v-if="showCloseButton"
        class="row">
        <button
          class="btn btn-secondary"
          @click="closeButtonClicked">
          {{ $t('buttons.cancel') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.container {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.row {
  margin-top: 20px;
  margin-left: 0;
  width: 100%;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background-color: white;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  width: 80%;
  max-width: 500px;
  overflow-y: auto;
  max-height: 90vh;
}
</style>
