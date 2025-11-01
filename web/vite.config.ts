import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'node:path'

export const ProjectRoot = resolve(import.meta.dirname, '../');
// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],

  build: {
    outDir: resolve(ProjectRoot, "dist/web",)
  },
})
