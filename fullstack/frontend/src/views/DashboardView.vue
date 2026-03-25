<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Activity, AlertTriangle, BriefcaseBusiness, FileText } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

import Badge from '@/components/ui/badge/Badge.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import Spinner from '@/components/ui/spinner/Spinner.vue'

import { api } from '../services/api'

const loading = ref(false)
const summary = reactive({
  role: '',
  scope: { institution: '', department: '', team: '' },
  candidates: 0,
  open_cases: 0,
  expiring_qualifications: 0,
  active_restrictions: 0,
})

const metricCards = computed(() => [
  {
    key: 'candidates',
    title: 'Candidates',
    subtitle: 'Talent pipeline records',
    value: summary.candidates,
    icon: BriefcaseBusiness,
    tone: 'text-primary',
  },
  {
    key: 'open_cases',
    title: 'Open Cases',
    subtitle: 'new + assigned + in_progress',
    value: summary.open_cases,
    icon: FileText,
    tone: 'text-secondary',
  },
  {
    key: 'expiring_qualifications',
    title: 'Expiring Qualifications',
    subtitle: 'Within next 30 days',
    value: summary.expiring_qualifications,
    icon: AlertTriangle,
    tone: 'text-warning',
  },
  {
    key: 'active_restrictions',
    title: 'Active Restrictions',
    subtitle: 'Controlled medication rules',
    value: summary.active_restrictions,
    icon: Activity,
    tone: 'text-success',
  },
])

async function loadSummary() {
  loading.value = true
  try {
    const data = await api.dashboard()
    Object.assign(summary, data)
  } catch (err) {
    toast.error(err.message || 'Failed to load dashboard')
  } finally {
    loading.value = false
  }
}

onMounted(loadSummary)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="section-title">Operational Snapshot</h2>
        <p class="section-subtitle">Realtime scope-aware metrics for your active role</p>
      </div>
      <Badge variant="outline">Role: {{ summary.role || '-' }}</Badge>
    </div>

    <div v-if="loading" class="rounded-xl border border-border bg-card p-4">
      <Spinner>Loading dashboard insights...</Spinner>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2 2xl:grid-cols-4">
      <Card v-for="card in metricCards" :key="card.key" class="hover:-translate-y-0.5 transition-transform duration-200">
        <CardHeader class="pb-2">
          <div class="flex items-center justify-between">
            <CardTitle class="text-sm">{{ card.title }}</CardTitle>
            <component :is="card.icon" class="h-4 w-4" :class="card.tone" />
          </div>
        </CardHeader>
        <CardContent>
          <p class="text-3xl font-bold tracking-tight text-foreground">{{ card.value }}</p>
          <p class="mt-1 text-xs text-muted-foreground">{{ card.subtitle }}</p>
        </CardContent>
      </Card>
    </div>

    <div class="panel-grid-2">
      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Current Scope</CardTitle>
        </CardHeader>
        <CardContent class="space-y-2 text-sm">
          <div class="flex justify-between gap-2">
            <span class="text-muted-foreground">Institution</span>
            <span class="font-medium text-foreground">{{ summary.scope.institution || '-' }}</span>
          </div>
          <div class="flex justify-between gap-2">
            <span class="text-muted-foreground">Department</span>
            <span class="font-medium text-foreground">{{ summary.scope.department || '-' }}</span>
          </div>
          <div class="flex justify-between gap-2">
            <span class="text-muted-foreground">Team</span>
            <span class="font-medium text-foreground">{{ summary.scope.team || '-' }}</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Platform Signals</CardTitle>
        </CardHeader>
        <CardContent class="space-y-2 text-sm">
          <div class="rounded-md bg-accent/70 p-2 text-accent-foreground">
            Qualification auto-deactivation is enforced for expired records.
          </div>
          <div class="rounded-md bg-muted p-2 text-muted-foreground">
            Case duplicate protection blocks similar submissions for 5 minutes.
          </div>
          <div class="rounded-md bg-secondary/10 p-2 text-secondary">
            Sensitive fields are encrypted at rest and masked in API responses.
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
