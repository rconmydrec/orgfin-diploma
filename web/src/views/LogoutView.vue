<script setup>
import { useRouter } from 'vue-router';
import { Services } from '../services/servicesConfig';
import { onMounted } from 'vue';

const router = useRouter();

onMounted(async () => {
  await Services.userService.logOutUser();
  await router.push({ name: 'home' });
});
</script>

<template>
  <div>
    <h1>Logging out...</h1>
  </div>
</template>
