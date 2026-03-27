<script setup>
import { computed, onMounted, reactive, ref } from "vue"
import { Archive, FilePlus2, Paperclip, RefreshCcw } from "lucide-vue-next"
import { toast } from "vue-sonner"

import Badge from "@/components/ui/badge/Badge.vue"
import Button from "@/components/ui/button/Button.vue"
import Card from "@/components/ui/card/Card.vue"
import CardContent from "@/components/ui/card/CardContent.vue"
import CardHeader from "@/components/ui/card/CardHeader.vue"
import CardTitle from "@/components/ui/card/CardTitle.vue"
import Input from "@/components/ui/input/Input.vue"
import Spinner from "@/components/ui/spinner/Spinner.vue"
import Textarea from "@/components/ui/textarea/Textarea.vue"

import { api } from "../services/api"

const loading = ref(false)
const uploadLoading = ref(false)
const uploadProgress = ref(0)
const historyLoading = ref(false)

const cases = ref([])
const attachments = ref([])
const caseHistory = ref([])
const selectedCaseId = ref("")
const selectedFile = ref(null)

const filters = reactive({
  status: "",
  q: "",
})

const caseForm = reactive({
  subject: "",
  description: "",
})

const assignForm = reactive({
  caseId: "",
  assignedTo: "",
})

const statusForm = reactive({
  caseId: "",
  status: "assigned",
})

const statusColumns = [
  { key: "new", label: "New", variant: "outline" },
  { key: "assigned", label: "Assigned", variant: "warning" },
  { key: "in_progress", label: "In Progress", variant: "default" },
  { key: "resolved", label: "Resolved", variant: "success" },
  { key: "closed", label: "Closed", variant: "secondary" },
]

const groupedCases = computed(() =>
  statusColumns.map((column) => ({
    ...column,
    items: cases.value.filter((item) => item.status === column.key),
  })),
)

function statusVariant(status) {
  const found = statusColumns.find((item) => item.key === status)
  return found?.variant || "outline"
}

async function loadCases() {
  loading.value = true
  try {
    cases.value = await api.listCases(filters)
    if (selectedCaseId.value) {
      await Promise.all([loadAttachments(), loadHistory()])
    }
  } catch (err) {
    toast.error(err.message || "Failed to load cases")
  } finally {
    loading.value = false
  }
}

async function createCase() {
  if (!caseForm.subject.trim() || !caseForm.description.trim()) {
    toast.warning("Subject and description are required")
    return
  }

  try {
    await api.createCase({ ...caseForm })
    toast.success("Case created")
    caseForm.subject = ""
    caseForm.description = ""
    await loadCases()
  } catch (err) {
    toast.error(err.message || "Failed to create case")
  }
}

async function assignCase() {
  if (!assignForm.caseId || !assignForm.assignedTo) {
    toast.warning("Case ID and assignee user ID are required")
    return
  }

  try {
    await api.assignCase(Number(assignForm.caseId), Number(assignForm.assignedTo))
    toast.success("Case assigned")
    await loadCases()
  } catch (err) {
    toast.error(err.message || "Failed to assign case")
  }
}

async function updateStatus() {
  if (!statusForm.caseId || !statusForm.status) {
    toast.warning("Case ID and status are required")
    return
  }

  try {
    await api.updateCaseStatus(Number(statusForm.caseId), statusForm.status)
    toast.success("Case status updated")
    await loadCases()
  } catch (err) {
    toast.error(err.message || "Failed to update case status")
  }
}

async function loadAttachments() {
  if (!selectedCaseId.value) return
  try {
    attachments.value = await api.listCaseAttachments(Number(selectedCaseId.value))
  } catch (err) {
    attachments.value = []
    toast.error(err.message || "Failed to load attachments")
  }
}

async function loadHistory() {
  if (!selectedCaseId.value) return
  historyLoading.value = true
  try {
    caseHistory.value = await api.listCaseHistory(Number(selectedCaseId.value))
  } catch (err) {
    caseHistory.value = []
    toast.error(err.message || "Failed to load case history")
  } finally {
    historyLoading.value = false
  }
}

function chooseCase(record) {
  selectedCaseId.value = String(record.id)
  assignForm.caseId = String(record.id)
  statusForm.caseId = String(record.id)
  loadAttachments()
  loadHistory()
}

function onFilePicked(event) {
  selectedFile.value = event.target.files?.[0] || null
}

async function uploadCaseAttachment() {
  if (!selectedCaseId.value || !selectedFile.value) {
    toast.warning("Select case and file first")
    return
  }

  uploadLoading.value = true
  uploadProgress.value = 0

  try {
    const file = selectedFile.value
    const chunkSize = 1024 * 1024
    const totalChunks = Math.ceil(file.size / chunkSize)

    const init = await api.uploadInit({
      module_name: "case_ledgers",
      record_id: Number(selectedCaseId.value),
      original_name: file.name,
      mime_type: file.type || "application/octet-stream",
      total_chunks: totalChunks,
      file_size: file.size,
    })

    for (let index = 0; index < totalChunks; index += 1) {
      const start = index * chunkSize
      const end = Math.min(start + chunkSize, file.size)
      const chunk = file.slice(start, end)
      await api.uploadChunk(init.upload_id, index, chunk, file.name)
      uploadProgress.value = Math.floor(((index + 1) / totalChunks) * 100)
    }

    await api.uploadComplete(init.upload_id)
    toast.success("Attachment uploaded successfully")
    selectedFile.value = null
    uploadProgress.value = 100
    await Promise.all([loadAttachments(), loadHistory()])
  } catch (err) {
    toast.error(err.message || "Attachment upload failed")
  } finally {
    uploadLoading.value = false
  }
}

onMounted(loadCases)
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <h2 class="section-title">Case Ledger</h2>
        <p class="section-subtitle">Create, assign, transition, and track full case processing history</p>
      </div>
      <Badge variant="outline">{{ cases.length }} case(s)</Badge>
    </div>

    <div class="panel-grid-2">
      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Create Case</CardTitle>
        </CardHeader>
        <CardContent class="space-y-3">
          <div>
            <label class="field-label">Subject</label>
            <Input v-model="caseForm.subject" placeholder="Case subject" />
          </div>
          <div>
            <label class="field-label">Description</label>
            <Textarea v-model="caseForm.description" :rows="3" placeholder="Sensitive case details" />
          </div>
          <Button @click="createCase">
            <FilePlus2 class="h-4 w-4" />
            Create Case
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Filters</CardTitle>
        </CardHeader>
        <CardContent class="space-y-3">
          <Input v-model="filters.q" placeholder="Search by case number or subject" @keyup.enter="loadCases" />
          <select v-model="filters.status" class="form-select" @change="loadCases">
            <option value="">All statuses</option>
            <option value="new">new</option>
            <option value="assigned">assigned</option>
            <option value="in_progress">in_progress</option>
            <option value="resolved">resolved</option>
            <option value="closed">closed</option>
          </select>
          <Button variant="outline" @click="loadCases">
            <RefreshCcw class="h-4 w-4" />
            Apply Filters
          </Button>
        </CardContent>
      </Card>
    </div>

    <div v-if="loading" class="rounded-xl border border-border bg-card p-4">
      <Spinner>Loading case boards...</Spinner>
    </div>

    <template v-else>
      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Kanban Status Board</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="grid grid-cols-1 gap-3 lg:grid-cols-5">
            <div v-for="column in groupedCases" :key="column.key" class="rounded-lg border border-border bg-muted/25 p-3">
              <div class="mb-3 flex items-center justify-between">
                <p class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{{ column.label }}</p>
                <Badge :variant="column.variant">{{ column.items.length }}</Badge>
              </div>

              <div class="space-y-2">
                <button
                  v-for="item in column.items"
                  :key="item.id"
                  type="button"
                  class="w-full rounded-md border border-border bg-card px-2 py-2 text-left transition-all hover:shadow-card"
                  @click="chooseCase(item)"
                >
                  <p class="text-xs font-semibold text-foreground">{{ item.case_no }}</p>
                  <p class="mt-1 line-clamp-2 text-xs text-muted-foreground">{{ item.subject }}</p>
                </button>

                <p v-if="!column.items.length" class="rounded-md border border-dashed border-border p-2 text-center text-xs text-muted-foreground">
                  No cases
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <div class="panel-grid-2">
        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Assign Case</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <Input v-model="assignForm.caseId" placeholder="Case ID" />
            <Input v-model="assignForm.assignedTo" placeholder="Assign to user ID" />
            <Button variant="outline" @click="assignCase">Assign</Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Status Transition</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <Input v-model="statusForm.caseId" placeholder="Case ID" />
            <select v-model="statusForm.status" class="form-select">
              <option value="assigned">assigned</option>
              <option value="in_progress">in_progress</option>
              <option value="resolved">resolved</option>
              <option value="closed">closed</option>
            </select>
            <Button variant="outline" @click="updateStatus">
              <Archive class="h-4 w-4" />
              Update Status
            </Button>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader class="pb-2">
          <CardTitle>Case Table</CardTitle>
        </CardHeader>
        <CardContent class="overflow-x-auto p-0">
          <table class="min-w-full divide-y divide-border text-sm">
            <thead class="bg-muted/60 text-left text-xs uppercase tracking-wide text-muted-foreground">
              <tr>
                <th class="px-4 py-3">ID</th>
                <th class="px-4 py-3">Case Number</th>
                <th class="px-4 py-3">Subject</th>
                <th class="px-4 py-3">Status</th>
                <th class="px-4 py-3">Assigned To</th>
                <th class="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <tr v-for="item in cases" :key="item.id" class="hover:bg-accent/40">
                <td class="px-4 py-3">{{ item.id }}</td>
                <td class="px-4 py-3 font-medium text-foreground">{{ item.case_no }}</td>
                <td class="px-4 py-3">{{ item.subject }}</td>
                <td class="px-4 py-3">
                  <Badge :variant="statusVariant(item.status)">{{ item.status }}</Badge>
                </td>
                <td class="px-4 py-3">{{ item.assigned_to_name || "-" }}</td>
                <td class="px-4 py-3 text-right">
                  <Button size="sm" variant="ghost" @click="chooseCase(item)">Open</Button>
                </td>
              </tr>
              <tr v-if="!cases.length">
                <td colspan="6" class="px-4 py-6 text-center text-muted-foreground">No cases found.</td>
              </tr>
            </tbody>
          </table>
        </CardContent>
      </Card>

      <div class="panel-grid-2">
        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Attachments</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <Input v-model="selectedCaseId" placeholder="Selected case ID" />
              <input
                type="file"
                class="block w-full text-sm text-muted-foreground file:mr-3 file:rounded-md file:border-0 file:bg-primary file:px-3 file:py-2 file:text-primary-foreground"
                @change="onFilePicked"
              />
            </div>

            <Button :loading="uploadLoading" @click="uploadCaseAttachment">
              <Paperclip class="h-4 w-4" />
              Upload Attachment
            </Button>

            <div class="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div class="h-full bg-secondary transition-all" :style="{ width: `${uploadProgress}%` }" />
            </div>

            <div class="overflow-x-auto rounded-lg border border-border">
              <table class="min-w-full divide-y divide-border text-sm">
                <thead class="bg-muted/60 text-left text-xs uppercase tracking-wide text-muted-foreground">
                  <tr>
                    <th class="px-4 py-3">ID</th>
                    <th class="px-4 py-3">Name</th>
                    <th class="px-4 py-3">MIME</th>
                    <th class="px-4 py-3">Size</th>
                    <th class="px-4 py-3">SHA256</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  <tr v-for="file in attachments" :key="file.id" class="hover:bg-accent/40">
                    <td class="px-4 py-3">{{ file.id }}</td>
                    <td class="px-4 py-3">{{ file.original_name }}</td>
                    <td class="px-4 py-3">{{ file.mime_type }}</td>
                    <td class="px-4 py-3">{{ file.file_size }}</td>
                    <td class="px-4 py-3 text-xs">{{ file.sha256 }}</td>
                  </tr>
                  <tr v-if="!attachments.length">
                    <td colspan="5" class="px-4 py-6 text-center text-muted-foreground">No attachments for this case.</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle>Case Processing History</CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <Input v-model="selectedCaseId" placeholder="Selected case ID" @keyup.enter="loadHistory" />
            <Button variant="outline" :loading="historyLoading" @click="loadHistory">Refresh History</Button>

            <div v-if="historyLoading" class="rounded-lg border border-border p-3 text-sm text-muted-foreground">
              Loading history...
            </div>
            <div v-else class="max-h-[340px] space-y-2 overflow-auto pr-1">
              <div v-for="entry in caseHistory" :key="entry.id" class="rounded-lg border border-border bg-muted/20 p-3 text-xs">
                <div class="flex items-center justify-between gap-2">
                  <p class="font-semibold text-foreground">{{ entry.action_type }}</p>
                  <p class="text-muted-foreground">{{ entry.created_at }}</p>
                </div>
                <p class="mt-1 text-muted-foreground">From: {{ entry.from_status || "-" }} → To: {{ entry.to_status || "-" }}</p>
                <p class="mt-1 text-muted-foreground">Assigned To: {{ entry.assigned_to || "-" }} • Changed By: {{ entry.changed_by }}</p>
                <p class="mt-1 text-muted-foreground">Note: {{ entry.note || "-" }}</p>
                <pre class="mt-2 overflow-auto rounded bg-background p-2 text-[11px] text-muted-foreground">{{ JSON.stringify(entry.details || {}, null, 2) }}</pre>
              </div>
              <div v-if="!caseHistory.length" class="rounded-lg border border-dashed border-border p-3 text-center text-xs text-muted-foreground">
                No history for this case.
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
