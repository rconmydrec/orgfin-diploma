<script setup>
import { onBeforeMount, ref } from 'vue';
import { useRoute } from 'vue-router';

import { Services } from '@/services/servicesConfig';

const route = useRoute();

const token = route.params.token;
const error = ref(false);
const errorMessage = ref('');
const loading = ref(true);

onBeforeMount(async () => {
  try {
    const activated = await Services.userService.activateUser(token);
    console.log(activated);
    loading.value = false;
  } catch (e) {
    console.log(e.message);
    errorMessage.value = e.message;
    loading.value = false;
    error.value = true;
  }
});
</script>

<template>
  <div class="container">
    <div class="row">
      <div class="col">
        <div
          v-if="loading"
          class="loading">
          <p>Loading...</p>
        </div>
        <div v-else>
          <div
            v-if="!error"
            class="alert alert-success">
            <p>{{ $t('message.accountActivated') }}</p>
          </div>
          <div
            v-if="error"
            class="alert alert-danger">
            <p>{{ $t('message.errorDuringActivation') }}: {{ errorMessage }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
@use '@/assets/main.scss' as *;
.loading {
  text-align: center;
  padding: 20px;
}

.error-msg {
  text-align: center;
  padding: 20px;
  color: red;
}
</style>
