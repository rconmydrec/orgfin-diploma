<script setup>
const props = defineProps(['text']);
</script>

<template>
  <div class="custom-hr">
    <div class="line"></div>
    <div class="text">{{ props.text }}</div>
    <div class="line"></div>
  </div>
</template>

<style scoped>
.custom-hr {
  display: flex;
  align-items: center;
  width: 100%;
}

.custom-hr .line {
  flex: 1;
  border-top: 1px solid #000;
}

.custom-hr .text {
  font-weight: bold;
  margin: 0 1rem;
}
</style>
