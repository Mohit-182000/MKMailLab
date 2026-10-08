import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import InboxView from '@/views/InboxView.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/inbox' },
  { path: '/inbox', name: 'inbox', component: InboxView },
  { path: '/config', name: 'config', component: () => import('@/views/ConfigView.vue') },
  { path: '/:pathMatch(.*)*', redirect: '/inbox' },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
