<script setup>
import { computed, onMounted, reactive, ref } from "vue"
import { Download, Search } from "lucide-vue-next"
import { toast } from "vue-sonner"

import Badge from "@/components/ui/badge/Badge.vue"
import Button from "@/components/ui/button/Button.vue"
import Card from "@/components/ui/card/Card.vue"
import CardContent from "@/components/ui/card/CardContent.vue"
import CardHeader from "@/components/ui/card/CardHeader.vue"
import CardTitle from "@/components/ui/card/CardTitle.vue"
import Input from "@/components/ui/input/Input.vue"
import Spinner from "@/components/ui/spinner/Spinner.vue"

import { api } from "../services/api"

const loading = ref(false)
const exportLoading = ref(false)
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)

const filters = reactive({
  module: "",
  action: "",
  category: "",
  level: "",
  q: "",
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size.value)))

function safeJSONString(value) {
  try {
    return JSON.stringify(value ?? null, null, 2)
  } catch {
    return String(value)
  }
}

function levelVariant(level) {
  if (level === "ERROR") return "danger"
  if (level === "WARN") return "warning"
  if (level === "INFO") return "default"
  return "outline"
}

async function loadLogs() {
  loading.value = true
  try {
    const data = await api.listAuditLogs({
      page: page.value,
      size: size.value,
      module: filters.module,
      action: filters.action,
      category: filters.category,
      level: filters.level,
      q: filters.q,
    })
    rows.value = data.items || []
    total.value = data.total || 0
  } catch (err) {
    toast.error(err.message || "Failed to load audit logs")
  } finally {
    loading.value = false
  }
}

async function exportCsv() {
  exportLoading.value = true
  try {
    const blob = await api.exportAuditCSV({
      module: filters.module,
      action: filters.action,
      category: filters.category,
      level: filters.level,
    })
    const url = URL.createObjectURL(blob)
    const link = document.createElement("a")
    link.href = url
    link.download = "audit_logs.csv"
    link.click()
    URL.revokeObjectURL(url)
    toast.success("Audit logs exported")
  } catch (err) {
    toast.error(err.message || "Failed to export audit logs")
  } finally {
    exportLoading.value = false
  }
}

function nextPage() {
  if (page.value >= totalPages.value) return
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
      <p class="section-subtitle">Append-only logs with structured before/after diffs, categories, and levels</p>
    </div>

    <Card>
      <CardHeader class="pb-2">
        <CardTitle>Filters</CardTitle>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid grid-cols-1 gap-2 md:grid-cols-5">
          <Input v-model="filters.module" placeholder="Module" />
          <Input v-model="filters.action" placeholder="Action" />
          <Input v-model="filters.category" placeholder="Category" />
          <Input v-model="filters.level" placeholder="Level" />
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
          <div class="space-y-2">
            <div v-for="row in rows" :key="row.id" class="rounded-lg border border-border bg-card p-3">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="flex flex-wrap items-center gap-2">
                  <Badge variant="outline">#{{ row.id }}</Badge>
                  <Badge :variant="levelVariant(row.level)">{{ row.level || "INFO" }}</Badge>
                  <Badge variant="secondary">{{ row.category || "general" }}</Badge>
                </div>
                <p class="text-xs text-muted-foreground">{{ row.created_at }}</p>
              </div>

              <div class="mt-2 grid grid-cols-1 gap-2 text-xs md:grid-cols-2">
                <p><span class="font-semibold">User:</span> {{ row.username || row.user_id }}</p>
                <p><span class="font-semibold">Action:</span> {{ row.action }}</p>
                <p><span class="font-semibold">Module:</span> {{ row.module_name }}</p>
                <p><span class="font-semibold">Record:</span> {{ row.record_id }}</p>
              </div>

              <div class="mt-3 grid grid-cols-1 gap-2 lg:grid-cols-2">
                <div>
                  <p class="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Before</p>
                  <pre class="max-h-44 overflow-auto rounded bg-muted/30 p-2 text-[11px]">{{ safeJSONString(row.before) }}</pre>
                </div>
                <div>
                  <p class="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">After</p>
                  <pre class="max-h-44 overflow-auto rounded bg-muted/30 p-2 text-[11px]">{{ safeJSONString(row.after) }}</pre>
                </div>
                <div>
                  <p class="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Diff</p>
                  <pre class="max-h-44 overflow-auto rounded bg-muted/30 p-2 text-[11px]">{{ safeJSONString(row.diff) }}</pre>
                </div>
                <div>
                  <p class="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Details</p>
                  <pre class="max-h-44 overflow-auto rounded bg-muted/30 p-2 text-[11px]">{{ safeJSONString(row.details) }}</pre>
                </div>
              </div>
            </div>

            <div v-if="!rows.length" class="rounded-lg border border-dashed border-border p-4 text-center text-muted-foreground">
              No audit records found.
            </div>
          </div>

          <div class="flex items-center justify-between">
            <p class="text-xs text-muted-foreground">
              Page {{ page }} / {{ totalPages }} • Showing up to {{ size }} entries • Total {{ total }}
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
