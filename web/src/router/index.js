import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { title: '价格总览', requiresAuth: true }
  },
  {
    path: '/products',
    name: 'Products',
    component: () => import('../views/Products.vue'),
    meta: { title: '商品管理', requiresAuth: true }
  },
  {
    path: '/trend',
    name: 'Trend',
    component: () => import('../views/Trend.vue'),
    meta: { title: '走势分析', requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  document.title = (to.meta.title ? to.meta.title + ' - ' : '') + '梦幻西游物价追踪'
  if (to.meta.requiresAuth) {
    const userStore = useUserStore()
    if (!userStore.isLoggedIn) {
      next({ path: '/login', query: { redirect: to.fullPath } })
      return
    }
  }
  if (to.path === '/login') {
    const userStore = useUserStore()
    if (userStore.isLoggedIn) {
      next('/')
      return
    }
  }
  next()
})

export default router
