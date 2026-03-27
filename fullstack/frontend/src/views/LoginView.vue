<script setup>
import { ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { FlaskConical } from "lucide-vue-next"
import { toast } from "vue-sonner"

import Card from "@/components/ui/card/Card.vue"
import CardContent from "@/components/ui/card/CardContent.vue"
import CardHeader from "@/components/ui/card/CardHeader.vue"
import CardTitle from "@/components/ui/card/CardTitle.vue"
import LoginForm from "@/components/common/LoginForm.vue"

import { api } from "../services/api"
import { setSession } from "../store/auth"

const router = useRouter()
const route = useRoute()
const loading = ref(false)

async function submitLogin(payload) {
  if (loading.value) return
  loading.value = true
  try {
    const data = await api.login({ username: payload.username, password: payload.password })
    setSession(data.access_token, data.user)
    toast.success("Login successful")
    router.push(route.query.redirect || "/dashboard")
  } catch (err) {
    toast.error(err.message || "Login failed")
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center px-4 py-10">
    <Card class="w-full max-w-lg border-white/30 bg-card/95 shadow-glass backdrop-blur">
      <CardHeader class="pb-1 text-center">
        <div class="mx-auto mb-4 inline-flex h-14 w-14 items-center justify-center rounded-xl bg-primary/10 text-primary">
          <FlaskConical class="h-6 w-6" />
        </div>
        <CardTitle class="text-2xl">Pharma Operations Platform</CardTitle>
        <p class="section-subtitle">Secure sign-in for compliance and talent workflows</p>
      </CardHeader>

      <CardContent class="space-y-4">
        <LoginForm @submit="submitLogin" />

        <p class="text-center text-xs text-muted-foreground">
          Credentials are managed by your administrator. JWT token duration: 8 hours.
        </p>
      </CardContent>
    </Card>
  </div>
</template>
