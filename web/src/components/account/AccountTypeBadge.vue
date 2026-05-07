<script setup>
import { computed } from 'vue';
import { ACCOUNT_TYPE_CONFIG } from '@/constants.js';

const props = defineProps({
  typeId: {
    type: Number,
    required: true,
  },
  showLabel: {
    type: Boolean,
    default: null,
  },
});

const config = computed(
  () => ACCOUNT_TYPE_CONFIG[props.typeId] || ACCOUNT_TYPE_CONFIG[1],
);

const badgeClass = computed(() => {
  const color = config.value.color;
  if (color === 'purple') {
    return 'border-purple text-purple';
  }
  return `border-${color} text-${color}`;
});
</script>

<template>
  <span
    class="badge rounded-pill account-type-badge"
    :class="badgeClass"
    :title="$t('accountTypes.' + config.key)">
    <i :class="config.icon"></i>
    <span
      v-if="showLabel === true"
      class="ms-1">
      {{ $t('accountTypes.' + config.key) }}
    </span>
    <span
      v-else-if="showLabel === null"
      class="d-none d-md-inline ms-1">
      {{ $t('accountTypes.' + config.key) }}
    </span>
  </span>
</template>

<style scoped>
.account-type-badge {
  font-size: 0.75rem;
  font-weight: 500;
  background-color: transparent;
}

.text-purple {
  color: #6f42c1 !important;
}

.border-purple {
  border-color: #6f42c1 !important;
}
</style>
