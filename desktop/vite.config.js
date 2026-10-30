import { defineConfig } from 'vite';
export default defineConfig({
  clearScreen: false,
  server: { fs: { allow: ['..'] } },
  worker: { format: 'iife' },
  build: { rollupOptions: { output: { manualChunks: { editor: ['monaco-editor/editor/editor.api.js'] } } } },
});
