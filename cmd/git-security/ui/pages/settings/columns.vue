<script setup lang="ts">
import { showConfirmationDialog } from '@/common-functions'

type ColumnType = 'string' | 'number' | 'boolean' | 'array' | 'date'
type ColumnConfig = {
  id: string
  type: ColumnType
  title: string
  description: string
  key: string
  width: number
  show: boolean
  filter: boolean
  filter_expanded?: boolean
  csv: boolean
  order: string
}

const collapsed = ref<Record<string, boolean>>({})

const columns = ref<ColumnConfig[]>([])
const fetchColumns = () => {
  $fetch("/api/v1/columns", {
    method: "GET",
    onResponse({ response }) {
      columns.value.splice(0)
      response._data.forEach((cc: ColumnConfig) => {
        columns.value.push(cc)
        const storedState = sessionStorage.getItem(cc.id)
        collapsed.value[cc.id] = storedState ? JSON.parse(storedState) : true
      })
    }
  })
}

const moved = (e: any) => {
  if (e.oldIndex != e.newIndex && columns.value.length > 1) {
    let currentIndex = e.newIndex - 1
    const movedColumn: ColumnConfig = columns.value[currentIndex]
    let prev = ""
    let next = ""
    if (currentIndex == 0) {
      next = columns.value[1].order
    } else if (currentIndex == columns.value.length - 1) {
      prev = columns.value[currentIndex - 1].order
    } else {
      prev = columns.value[currentIndex - 1].order
      next = columns.value[currentIndex + 1].order
    }
    $fetch("/api/v1/columns/order", {
      method: "POST",
      body: {
        id: movedColumn.id,
        prev: prev,
        next: next
      },
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: 'Success',
            message: 'Order was changed successfully',
            type: 'success',
            position: 'bottom-right'
          })
        } else {
          ElNotification({
            title: 'Error',
            message: 'Internal error occurred',
            type: 'error',
            position: 'bottom-right'
          })
        }
        fetchColumns()
      }
    })
  }
}

const columnChanged = (index: number) => {
  setTimeout(() => {
    const c = columns.value[index]
    $fetch(`/api/v1/column/${c.id}`, {
      method: "PUT",
      body: c,
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: 'Success',
            message: 'Column config was updated successfully',
            type: 'success',
            position: 'bottom-right'
          })
        } else {
          ElNotification({
            title: 'Error',
            message: 'Internal error occurred',
            type: 'error',
            position: 'bottom-right'
          })
        }
        fetchColumns()
      }
    })
  }, 500)
}

const deleteColumn = async (id: string, column: string) => {
  try {
    const confirmed = await showConfirmationDialog(`Are you sure you want to delete the column: \n${column} ?`)
    if (confirmed) {
      $fetch(`/api/v1/column/${id}`, {
        method: "DELETE",
        onResponse({ response }) {
          if (response.status == 200) {
            ElNotification({
              title: 'Success',
              message: 'Column config was deleted successfully',
              type: 'success',
              position: 'bottom-right'
            })
          } else {
            ElNotification({
              title: 'Error',
              message: 'Internal error occurred',
              type: 'error',
              position: 'bottom-right'
            })
          }
          fetchColumns()
        }
      })

    }
  } catch (error) {
    console.error(error)
  }
}

const allKnownDataKeys = ref<DataKeyItem[]>([
  { value: "created_at", link: "" },
  { value: "default_branch.name", link: "" },
  { value: "default_branch.branch_protection_rule.pattern", link: "" },
  { value: "default_branch.branch_protection_rule.allows_deletion", link: "" },
  { value: "default_branch.branch_protection_rule.allows_force_pushes", link: "" },
  { value: "default_branch.branch_protection_rule.dismisses_stale_reviews", link: "" },
  { value: "default_branch.branch_protection_rule.is_admin_enforced", link: "" },
  { value: "default_branch.branch_protection_rule.require_last_push_approval", link: "" },
  { value: "default_branch.branch_protection_rule.required_approving_review_count", link: "" },
  { value: "default_branch.branch_protection_rule.required_status_checks", link: "" },
  { value: "default_branch.branch_protection_rule.requires_approving_reviews", link: "" },
  { value: "default_branch.branch_protection_rule.requires_code_owner_reviews", link: "" },
  { value: "default_branch.branch_protection_rule.requires_commit_signatures", link: "" },
  { value: "default_branch.branch_protection_rule.requires_conversation_resolution", link: "" },
  { value: "default_branch.branch_protection_rule.requires_linear_history", link: "" },
  { value: "default_branch.branch_protection_rule.requires_status_checks", link: "" },
  { value: "default_branch.branch_protection_rule.requires_strict_status_checks", link: "" },
  { value: "default_branch.branch_protection_rule.retricts_pushes", link: "" },
  { value: "default_branch.branch_protection_rule.retricts_review_dismissals", link: "" },
  { value: "delete_branch_on_merge", link: "" },
  { value: "disk_usage", link: "" },
  { value: "full_name", link: "" },
  { value: "is_archived", link: "" },
  { value: "is_disabled", link: "" },
  { value: "is_empty", link: "" },
  { value: "is_locked", link: "" },
  { value: "is_private", link: "" },
  { value: "last_committed_at", link: "" },
  { value: "merge_commit_allowed", link: "" },
  { value: "name", link: "" },
  { value: "owner.login", link: "" },
  { value: "primary_language.name", link: "" },
  { value: "pull_requests.total_count", link: "" },
  { value: "rebase_merge_allowed", link: "" },
  { value: "refs.total_count", link: "" },
  { value: "squash_merge_allowed", link: "" },
  { value: "updated_at", link: "" }
])

interface DataKeyItem {
  value: string
  link: string
}
const createFilter = (queryString: string) => {
  return (s: DataKeyItem) => {
    return (
      s.value.indexOf(queryString) >= 0
    )
  }
}

const columnDataKeys = (queryString: string, cb: any) => {
  const results = queryString
    ? allKnownDataKeys.value.filter(createFilter(queryString))
    : allKnownDataKeys.value
  // call callback function to return suggestions
  cb(results)
}

const addColumn = () => {
  $fetch("/api/v1/columns", {
    method: "POST",
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: 'Success',
          message: 'New column config was added successfully',
          type: 'success',
          position: 'bottom-right'
        })
      } else {
        ElNotification({
          title: 'Error',
          message: 'Internal error occurred',
          type: 'error',
          position: 'bottom-right'
        })
      }
      fetchColumns()
    }
  })
}

const toggleCollapse = (id: string) => {
  collapsed.value[id] = !collapsed.value[id]
  sessionStorage.setItem(id, JSON.stringify(collapsed.value[id]))
}

onMounted(() => {
  fetchColumns()
})
</script>

<template>
  <draggable tag="div"
             :list="columns"
             handle=".handle"
             item-key="id"
             @end="moved">
    <template #header>
      <div class="add-button">
        <UButton icon="i-fa6-solid-plus"
                 color="gray"
                 variant="ghost"
                 aria-label="Theme"
                 @click="addColumn" />
      </div>
    </template>
    <template #item="{ element, index }">
      <el-card shadow="always"
               class="column">
        <template #header>
          <div class="card-header">
            <div>
              <span class="title">#{{ index + 1 }} {{ element.title }}</span>
              <el-checkbox v-model="element.show"
                           label="Show in table?"
                           class="m-2 m-3"
                           size="large"
                           border
                           @change="columnChanged(index)" />
              <el-checkbox v-model="element.filter"
                           label="Show in filters?"
                           class="m-2"
                           size="large"
                           border
                           v-if="element.type != 'date'"
                           @change="columnChanged(index)" />
              <el-checkbox v-model="element.filter_expanded"
                           label="Expanded in filters?"
                           class="m-2"
                           size="large"
                           border
                           v-if="element.type != 'date'"
                           @change="columnChanged(index)" />
              <el-checkbox v-model="element.csv"
                           label="Included in exported CSV?"
                           class="m-2"
                           size="large"
                           border
                           @change="columnChanged(index)" />
              <el-button type="danger"
                         class="delete-button"
                         circle
                         plain
                         @click="deleteColumn(element.id, element.title)">
                <UIcon name="i-fa6-solid-trash-can" />
              </el-button>
            </div>
            <div>
              <UIcon :name="collapsed[element.id] ? 'i-fa6-solid-chevron-down' : 'i-fa6-solid-chevron-up'"
                     class="handle"
                     @click="toggleCollapse(element.id)" />
              <UIcon name="i-fa6-solid-align-justify"
                     class="handle" />
            </div>
          </div>
        </template>
        <div v-show="!collapsed[element.id]">
          <el-input v-model="element.title"
                    class="w-30 m-2"
                    :placeholder="element.title"
                    size="large"
                    @change="columnChanged(index)">
            <template #prepend>Title</template>
          </el-input>
          <el-autocomplete v-model="element.key"
                           :fetch-suggestions="columnDataKeys"
                           class="m-2"
                           size="large"
                           @change="columnChanged(index)"
                           :style="{ width: '30%' }">
            <template #prepend>Data Key</template>
          </el-autocomplete>
          <el-input v-model.number="element.width"
                    type="number"
                    class="w-30 m-2"
                    size="large"
                    @change="columnChanged(index)">
            <template #prepend>Column Width</template>
          </el-input>
        </div>
        <div v-show="!collapsed[element.id]">
          <el-select v-model="element.type"
                     class="w-30 m-2"
                     placeholder="Column Type"
                     size="large"
                     @change="columnChanged(index)">
            <template #prefix>Data Type</template>
            <el-option key="string"
                       label="String"
                       value="string" />
            <el-option key="number"
                       label="Number"
                       value="number" />
            <el-option key="boolean"
                       label="Boolean"
                       value="boolean" />
            <el-option key="array"
                       label="Array"
                       value="array" />
            <el-option key="date"
                       label="Date"
                       value="date" />
          </el-select>
          <el-input v-model="element.description"
                    class="w-60 m-2"
                    :placeholder="element.description"
                    size="large"
                    @change="columnChanged(index)">
            <template #prepend>Description</template>
          </el-input>
        </div>
      </el-card>
    </template>
  </draggable>
</template>

<style scoped>
.add-button {
  text-align: right;
  margin: 10px;
}

.delete-button {
  float: top;
  margin-bottom: 5px;
  margin-left: 9px;
}

.column {
  margin-bottom: 10px;
}

.handle {
  float: right;
  margin-top: 1px;
  margin-right: 10px;
  cursor: pointer;
}

.w-30 {
  width: 30%;
}

.w-60 {
  width: 61%;
}

.m-2 {
  margin: 0.5rem;
}

.card-header {
  display: flex;
  font-weight: bold;
  align-items: center;
  justify-content: space-between;
  margin-left: 5px;
}

.el-card {
  --el-card-padding: 0px;
}

.m-3 {
  margin-left: 30px;
}

.title {
  width: 100px;
}
</style>