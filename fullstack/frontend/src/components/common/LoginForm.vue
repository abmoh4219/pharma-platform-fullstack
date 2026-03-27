<script setup>
import { reactive, ref } from "vue"
import { LockKeyhole, User } from "lucide-vue-next"

import Button from "@/components/ui/button/Button.vue"
import Input from "@/components/ui/input/Input.vue"

const emit = defineEmits(["submit"])

const form = reactive({
  username: "",
  password: "",
})

const submitting = ref(false)

async function handleSubmit() {
  if (submitting.value) return
  submitting.value = true
  try {
    await emit("submit", { ...form })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="space-y-4" @submit.prevent="handleSubmit">
    <div class="space-y-2">
      <label class="text-sm font-medium text-foreground" for="username">Username</label>
      <div class="relative">
        <User class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input id="username" v-model="form.username" class="pl-9" placeholder="Enter username" />
      </div>
    </div>

    <div class="space-y-2">
      <label class="text-sm font-medium text-foreground" for="password">Password</label>
      <div class="relative">
        <LockKeyhole class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input id="password" v-model="form.password" type="password" class="pl-9" placeholder="Enter password" />
      </div>
    </div>

    <Button type="submit" class="w-full" :loading="submitting" data-testid="login-submit">
      Sign In
    </Button>
  </form>
</template>
