import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import DashboardLayout from '../components/layout/DashboardLayout.vue'
import OverviewView from '../views/OverviewView.vue'
import NodesView from '../views/NodesView.vue'
import TrafficView from '../views/TrafficView.vue'
import ACLView from '../views/ACLView.vue'
import AlertsView from '../views/AlertsView.vue'
import SettingsView from '../views/SettingsView.vue'
import { useAuthStore } from '../stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: LoginView,
  },
  {
    path: '/',
    component: DashboardLayout,
    meta: { requiresAuth: true },
    children: [
      { path: '', name: 'overview', component: OverviewView },
      { path: 'nodes', name: 'nodes', component: NodesView },
      { path: 'traffic', name: 'traffic', component: TrafficView },
      { path: 'acl', name: 'acl', component: ACLView },
      { path: 'alerts', name: 'alerts', component: AlertsView },
      { path: 'settings', name: 'settings', component: SettingsView },
    ],
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login' }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'overview' }
  }
  return true
})
