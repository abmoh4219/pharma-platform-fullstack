<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ClipboardList,
  FolderKanban,
  LayoutDashboard,
  LogOut,
  Menu,
  ShieldCheck,
  Stethoscope,
  UsersRound,
  X,
} from 'lucide-vue-next'
import { Toaster, toast } from 'vue-sonner'

import Badge from '@/components/ui/badge/Badge.vue'
import Button from '@/components/ui/button/Button.vue'
import ConfirmDialog from '@/components/ui/dialog/ConfirmDialog.vue'

import { api } from './services/api'
import { authState, clearSession, roleMenus } from './store/auth'

const route = useRoute()
const router = useRouter()

const sidebarOpen = ref(false)
const logoutDialogOpen = ref(false)
const logoutLoading = ref(false)

const isLoginPage = computed(() => route.path === '/login')

const roleLabel = computed(() =>
  (authState.user?.role || '-')
    .split('_')
    .map((value) => value.charAt(0).toUpperCase() + value.slice(1))
    .join(' '),
)

const iconByPath = {
  '/dashboard': LayoutDashboard,
  '/recruitment': UsersRound,
  '/compliance': ShieldCheck,
  '/cases': FolderKanban,
  '/audit': ClipboardList,
}

const menus = computed(() =>
  roleMenus(authState.user?.role).map((item) => ({
    ...item,
    icon: iconByPath[item.key] || LayoutDashboard,
  })),
)

const pageTitle = computed(() => route.meta?.title || 'Pharma Operations')

watch(
  () => route.fullPath,
  () => {
    sidebarOpen.value = false
  },
)

function navigate(path) {
  if (path !== route.path) {
    router.push(path)
  }
}

async function onLogoutConfirm() {
  if (logoutLoading.value) return
  logoutLoading.value = true
  try {
    await api.logout()
  } catch {
    // intentionally ignored; local session is still cleared
  } finally {
    clearSession()
    logoutDialogOpen.value = false
    logoutLoading.value = false
    toast.success('Logged out successfully')
    router.push('/login')
  }
}
</script>

<template>
  <Toaster rich-colors position="top-right" />

  <router-view v-if="isLoginPage" />

  <div v-else class="min-h-screen">
    <div v-if="sidebarOpen" class="fixed inset-0 z-40 bg-black/45 lg:hidden" @click="sidebarOpen = false" />

    <aside
      class="fixed inset-y-0 left-0 z-50 w-72 border-r border-border bg-card/95 px-4 pb-4 pt-5 shadow-glass backdrop-blur"
      :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'"
    >
      <div class="flex items-center justify-between rounded-xl bg-accent/70 px-3 py-2">
        <div class="flex items-center gap-2">
          <div class="inline-flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-card">
            <Stethoscope class="h-5 w-5" />
          </div>
          <div>
            <p class="text-sm font-semibold text-foreground">Pharma Platform</p>
            <p class="text-xs text-muted-foreground">Compliance + Talent Ops</p>
          </div>
        </div>
        <button class="rounded-md p-1 text-muted-foreground hover:bg-background lg:hidden" @click="sidebarOpen = false">
          <X class="h-4 w-4" />
        </button>
      </div>

      <nav class="mt-5 space-y-1.5">
        <button
          v-for="menu in menus"
          :key="menu.key"
          type="button"
          class="group flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-left text-sm transition-all"
          :class="
            route.path === menu.key
              ? 'bg-primary text-primary-foreground shadow-card'
              : 'text-foreground hover:bg-accent/70'
          "
          @click="navigate(menu.key)"
        >
          <span class="flex items-center gap-2">
            <component :is="menu.icon" class="h-4 w-4" />
            {{ menu.label }}
          </span>
          <span
            class="h-1.5 w-1.5 rounded-full"
            :class="route.path === menu.key ? 'bg-primary-foreground' : 'bg-transparent group-hover:bg-secondary'"
          />
        </button>
      </nav>

      <div class="mt-6 rounded-lg border border-border bg-muted/40 p-3 text-xs text-muted-foreground">
        Data scope enforcement is active for every protected API call.
      </div>
    </aside>

    <div class="min-h-screen lg:pl-72">
      <header class="sticky top-0 z-30 border-b border-border bg-background/85 backdrop-blur">
        <div class="page-container flex items-center justify-between gap-3 py-3">
          <div class="flex items-center gap-3">
            <button
              type="button"
              class="rounded-md border border-border bg-card p-2 text-foreground hover:bg-accent lg:hidden"
              @click="sidebarOpen = true"
            >
              <Menu class="h-4 w-4" />
            </button>
            <div>
              <p class="text-base font-semibold text-foreground">{{ pageTitle }}</p>
              <p class="text-xs text-muted-foreground">Pharmaceutical Compliance & Talent Operations Platform</p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <Badge variant="outline">{{ authState.user?.username || 'unknown' }}</Badge>
            <Badge variant="success">{{ roleLabel }}</Badge>
            <Button variant="outline" size="sm" @click="logoutDialogOpen = true">
              <LogOut class="h-4 w-4" />
              Logout
            </Button>
          </div>
        </div>
      </header>

      <main class="page-container">
        <router-view />
      </main>
    </div>
  </div>

  <ConfirmDialog
    v-model:open="logoutDialogOpen"
    title="Confirm logout"
    description="Your token will be invalidated and you will return to the login page."
    confirm-text="Logout"
    cancel-text="Stay"
    :loading="logoutLoading"
    @confirm="onLogoutConfirm"
  />
</template>
