import { RouteRecordRaw, createMemoryHistory, createRouter as createVueRouter, createWebHistory } from 'vue-router'
import HomePage from './pages/HomePage.vue'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: HomePage,
  },
]

export function createRouter() {
  const history = import.meta.env.SSR
    ? createMemoryHistory(import.meta.env.BASE_URL)
    : createWebHistory(import.meta.env.BASE_URL)

  return createVueRouter({
    history,
    routes,
  })
}
