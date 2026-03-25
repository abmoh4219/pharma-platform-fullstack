<script setup>
import { onMounted, reactive, ref } from 'vue'
import { Download, Search } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import Input from '@/components/ui/input/Input.vue'
import Spinner from '@/components/ui/spinner/Spinner.vue'

import { api } from '../services/api'

const loading = ref(false)
const exportLoading = ref(false)
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)

const filters = reactive({
  module: '',
  action: '',
  q: '',
})

async function loadLogs() {
  loading.value = true
  try {
    const data = await api.listAuditLogs({
      page: page.value,
      size: size.value,
      module: filters.module,
      action: filters.action,
      q: filters.q,
    })
    rows.value = data.items || []
    total.value = data.total || 0
  } catch (err) {
    toast.error(err.message || 'Failed to load audit logs')
  } finally {
    loading.value = false
  }
}

async function exportCsv() {
  exportLoading.value = true
  try {
    const blob = await api.exportAuditCSV({ module: filters.module, action: filters.action })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'audit_logs.csv'
    link.click()
    URL.revokeObjectURL(url)
    toast.success('Audit logs exported')
  } catch (err) {
    toast.error(err.message || 'Failed to export audit logs')
  } finally {
    exportLoading.value = false
  }
}

function nextPage() {
  if (page.value * size.value >= total.value) return
  page.value += 1
  loadLogs()
}

function previousPage() {
  if (page.value <= 1) return
  page.value -= 1
  loadLogs()
}

onMounted(loadLogs)
</script>

<template>
  <div class="space-y-4">
    <div>
      <h2 class="section-title">Audit Trail</h2>
      <p class="section-subtitle">Append-only logs for all sensitive operations</p>
    </div>

    <Card>
      <CardHeader class="pb-2">
        <CardTitle>Filters</CardTitle>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid grid-cols-1 gap-2 md:grid-cols-3">
          <Input v-model="filters.module" placeholder="Module" />
          <Input v-model="filters.action" placeholder="Action" />
          <Input v-model="filters.q" placeholder="Search details" @keyup.enter="loadLogs" />
        </div>

        <div class="flex flex-wrap gap-2">
          <Button @click="loadLogs">
            <Search class="h-4 w-4" />
            Search
          </Button>
          <Button variant="outline" :loading="exportLoading" @click="exportCsv">
            <Download class="h-4 w-4" />
            Export CSV
          </Button>
        </div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="pb-2">
        <CardTitle>Audit Logs</CardTitle>
      </CardHeader>
      <CardContent>
        <div v-if="loading" class="py-2">
          <Spinner>Loading audit logs...</Spinner>
        </div>

        <div v-else class="space-y-3">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-border text-sm">
              <thead class="bg-muted/60 text-left text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th class="px-3 py-2">ID</th>
                  <th class="px-3 py-2">Time</th>
                  <th class="px-3 py-2">User</th>
                  <th class="px-3 py-2">Action</th>
                  <th class="px-3 py-2">Module</th>
                  <th class="px-3 py-2">Record</th>
                  <th class="px-3 py-2">Details</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr v-for="row in rows" :key="row.id" class="hover:bg-accent/40">
                  <td class="px-3 py-2">{{ row.id }}</td>
                  <td class="px-3 py-2">{{ row.created_at }}</td>
                  <td class="px-3 py-2">{{ row.username }}</td>
                  <td class="px-3 py-2">{{ row.action }}</td>
                  <td class="px-3 py-2">{{ row.module_name }}</td>
                  <td class="px-3 py-2">{{ row.record_id }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ row.details }}</td>
                </tr>
                <tr v-if="!rows.length">
                  <td colspan="7" class="px-3 py-6 text-center text-muted-foreground">No audit records found.</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="flex items-center justify-between">
            <p class="text-xs text-muted-foreground">
              Page {{ page }} • Showing up to {{ size }} entries • Total {{ total }}
            </p>
            <div class="flex gap-2">
              <Button variant="outline" size="sm" @click="previousPage">Previous</Button>
              <Button variant="outline" size="sm" @click="nextPage">Next</Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
