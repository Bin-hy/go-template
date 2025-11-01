import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: () => import('../pages/Home.vue') },
  { path: '/health', name: 'health', component: () => import('../pages/Health.vue') },
  { path: '/files', name: 'files', component: () => import('../pages/Files.vue') },
  { path: '/upload-chunks', name: 'upload-chunks', component: () => import('../pages/ChunkUpload.vue') },
  { path: '/swagger', name: 'swagger', component: () => import('../pages/Swagger.vue') },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})