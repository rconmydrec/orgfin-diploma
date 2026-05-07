<script setup>
import { ref } from 'vue';

const props = defineProps(['itemType', 'transaction', 'isEdit']);
const emit = defineEmits(['typeChanged']);

const currentItem = ref(props.itemType);

function changeItemType($event) {
  currentItem.value = $event.target.getAttribute('data-itemtype');
  emit('typeChanged', currentItem.value);
}
</script>

<template>
  <ul
    @click="changeItemType"
    class="nav nav-pills bottom-space top-space"
    v-if="!isEdit || (isEdit && !transaction.isTransfer)">
    <li class="nav-item">
      <a
        id="expense-item"
        class="nav-link"
        :class="{ active: props.itemType === 'expense' }"
        data-itemtype="expense"
        href="#">
        {{ $t('message.expense') }}
      </a>
    </li>
    <li class="nav-item">
      <a
        id="income-item"
        class="nav-link"
        :class="{ active: props.itemType === 'income' }"
        data-itemtype="income"
        href="#">
        {{ $t('message.income') }}
      </a>
    </li>
    <li class="nav-item">
      <a
        id="transfer-item"
        class="nav-link"
        :class="props.itemType === 'transfer' ? 'active' : ''"
        data-itemtype="transfer"
        href="#">
        {{ $t('message.transfer') }}
      </a>
    </li>
  </ul>
</template>
