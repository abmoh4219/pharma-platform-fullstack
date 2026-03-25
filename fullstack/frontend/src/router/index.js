import { createRouter, createWebHistory } from 'vue-router'

import { hasAnyRole, isAuthenticated } from '../store/auth'

import LoginView from '../views/LoginView.vue'
import DashboardView from '../views/DashboardView.vue'
import RecruitmentView from '../views/RecruitmentView.vue'
import ComplianceView from '../views/ComplianceView.vue'
import CasesView from '../views/CasesView.vue'
import AuditView from '../views/AuditView.vue'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: { public: true, title: 'Sign In' },
  },
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: DashboardView,
    meta: { auth: true, title: 'Operations Dashboard' },
  },
  {
    path: '/recruitment',
    name: 'recruitment',
    component: RecruitmentView,
    meta: { auth: true, title: 'Recruitment Operations', roles: ['recruitment_specialist', 'system_admin'] },
  },
  {
    path: '/compliance',
    name: 'compliance',
    component: ComplianceView,
    meta: { auth: true, title: 'Compliance Operations', roles: ['compliance_admin', 'system_admin'] },
  },
  {
    path: '/cases',
    name: 'cases',
    component: CasesView,
    meta: {
      auth: true,
      title: 'Case Ledger Operations',
      roles: ['business_specialist', 'compliance_admin', 'recruitment_specialist', 'system_admin'],
    },
  },
  {
    path: '/audit',
    name: 'audit',
    component: AuditView,
    meta: { auth: true, title: 'Audit Logs', roles: ['compliance_admin', 'system_admin'] },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  if (to.meta.public) {
    return true
  }
  if (to.meta.auth && !isAuthenticated()) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.roles && !hasAnyRole(to.meta.roles)) {
    return { path: '/dashboard' }
  }
  return true
})

export default router
