import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/intranet/',
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/admin': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/bundle.js': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/components': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
