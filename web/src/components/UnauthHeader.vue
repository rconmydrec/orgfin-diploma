<script setup>
import { RouterLink } from 'vue-router';
import LanguageSwitcher from './LanguageSwitcher.vue';
</script>

<template>
  <div class="unauth-header">
    <RouterLink
      :to="{ name: 'home' }"
      class="logo"
      aria-label="Orgfin home">
      <img
        src="/images/logo.png"
        alt="Orgfin"
        class="logo-img" />
    </RouterLink>

    <LanguageSwitcher />
  </div>
</template>

<style scoped>
.unauth-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  z-index: 1000;
}

.logo {
  text-decoration: none;
  line-height: 1;
}

.logo-img {
  display: block;
  height: 28px;
  width: auto;
}
</style>
