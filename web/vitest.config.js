import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';

// Vitest config for unit / component tests.
// Kept separate from `vite.config.js` so build pipelines stay byte-identical.
export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.{js,mjs}'],
    // Most test specs do not exercise SCSS — disable processing to keep the
    // suite fast and avoid pulling in sass/bootstrap during unit runs.
    css: false,
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
});
