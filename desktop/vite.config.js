import { defineConfig } from 'vite';
import { cpSync, createReadStream } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const projectDir = path.dirname(fileURLToPath(import.meta.url));
const mathLiveDir = path.join(projectDir, 'node_modules', 'mathlive');
const mathLiveAssets = {
  name: 'margo-mathlive-assets',
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      const prefix = '/mathlive-assets/';
      if (!req.url?.startsWith(prefix)) return next();
      const relative = decodeURIComponent(req.url.slice(prefix.length).split('?')[0]);
      if (!relative.startsWith('fonts/') && !relative.startsWith('sounds/')) return next();
      const assetPath = path.resolve(mathLiveDir, relative);
      if (!assetPath.startsWith(mathLiveDir + path.sep)) return next();
      const extension = path.extname(assetPath);
      res.setHeader('Content-Type', extension === '.woff2' ? 'font/woff2' : extension === '.wav' ? 'audio/wav' : 'application/octet-stream');
      createReadStream(assetPath).on('error', next).pipe(res);
    });
  },
  closeBundle() {
    for (const directory of ['fonts', 'sounds']) {
      cpSync(path.join(mathLiveDir, directory), path.join(projectDir, 'dist', 'mathlive-assets', directory), { recursive: true });
    }
  },
};

export default defineConfig({
  clearScreen: false,
  server: { fs: { allow: ['..'] } },
  worker: { format: 'iife' },
  plugins: [mathLiveAssets],
  build: { rollupOptions: { output: { manualChunks: { editor: ['monaco-editor/editor/editor.api.js'] } } } },
});
