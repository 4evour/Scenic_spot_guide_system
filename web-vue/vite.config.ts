import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig(({ command }) => ({
  base: command === 'build' ? '/static/vue-app/' : '/',
  plugins: [vue()],
  build: {
    outDir: '../static/vue-app',
    emptyOutDir: true,
    sourcemap: false,
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/static': 'http://127.0.0.1:8080',
      '/v1': 'http://127.0.0.1:8080',
      '/vtuber-ws': {
        target: 'ws://127.0.0.1:12393',
        ws: true,
        rewrite: path => path.replace(/^\/vtuber-ws/, ''),
      },
    },
  },
}));
