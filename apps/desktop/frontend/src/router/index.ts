import { createRouter, createWebHashHistory } from 'vue-router'
import { tokenStorage } from '../api/token-storage'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('../layouts/MainLayout.vue'),
      children: [
        { path: '', redirect: '/projects' },
        { path: 'projects', name: 'projects', component: () => import('../views/ProjectsView.vue') },
        { path: 'projects/:id', name: 'project', component: () => import('../views/ProjectView.vue') },
        { path: 'settings', name: 'settings', component: () => import('../views/SettingsView.vue') },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.initialized) {
    await auth.loadMe()
  }
  const authenticated = Boolean(tokenStorage.getAccess())
  if (!to.meta.public && !authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && authenticated) {
    return { name: 'projects' }
  }
  return true
})

export default router