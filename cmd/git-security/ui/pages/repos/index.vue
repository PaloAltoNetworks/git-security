<script setup lang="ts">
import {
  ElCheckbox,
  ElLink,
  ElNotification,
  TableV2FixedDir,
  TableV2SortOrder
} from 'element-plus'
import type { CheckboxValueType, Column, RowClassNameGetter, SortBy } from 'element-plus'
import { Loading as LoadingIcon } from '@element-plus/icons-vue'
import Icon from '#ui/components/elements/Icon.vue'

type ColumnType = 'string' | 'number' | 'boolean'
type ColumnConfig = {
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

const loading = ref(false)
const filters = ref<Record<string, string[]>>({})
const filtersOrder = ref([])
const updateFilters = (field: string) => {
  fetchRepos()
}

const transform = (f: Record<string, string[]>) => {
  let results = []
  for (let filter in f) {
    if (f[filter].length > 0) {
      results.push({
        field: filter,
        values: f[filter],
      })
    }
  }
  return results
}

const fetchRepos = () => {
  loading.value = true
  useFetch("/api/v1/repos", {
    method: "POST",
    body: {
      filters: transform(filters.value)
    },
    onResponse({ response }) {
      loading.value = false
      repos_table.data = response._data
      repos_table.data = useOrderBy(
        repos_table.data,
        ["full_name"],
        [TableV2SortOrder.ASC],
      )
      repos_table.selected.clear()
    }
  })
}

const refreshFilter = async () => {
  Object.keys(filters.value).forEach(key => delete filters.value[key]);
  filtersOrder.value.splice(0)
  fetchRepos()
}

const exportToCSV = async () => {
  useFetch('/api/v1/repos?csv=true', {
    method: "POST",
    body: {
      filters: transform(filters.value)
    },
    onResponse({ response }) {
      const url = "data:text/csv;charset=utf-8," + encodeURIComponent(response._data);
      const a = document.createElement("a");
      a.href = url;
      a.setAttribute("download", "repos.csv");
      a.click();
    }
  })
}

const columns: Column<any>[] = [
  {
    key: 'selection',
    width: 25,
    cellRenderer: ({ rowData }) => {
      const onChange = (value: CheckboxValueType) => {
        if (value) {
          repos_table.selected.set(rowData['id'], true)
        } else {
          repos_table.selected.delete(rowData['id'])
        }
      }
      return h(
        ElCheckbox,
        {
          onChange: onChange,
          modelValue: repos_table.selected.has(rowData['id'])
        }
      )
    },
    headerCellRenderer: () => {
      const onChange = (value: CheckboxValueType) => {
        for (var i in repos_table.data) {
          const rowData = repos_table.data[i]
          if (value) {
            repos_table.selected.set(rowData['id'], true)
          } else {
            repos_table.selected.delete(rowData['id'])
          }
        }
      }
      return h(
        ElCheckbox,
        {
          modelValue: repos_table.selected.size > 0 && repos_table.selected.size == repos_table.data.length,
          indeterminate: repos_table.selected.size > 0 && repos_table.selected.size != repos_table.data.length,
          onChange: onChange
        }
      )
    },
    "fixed": TableV2FixedDir.LEFT,
  },
  {
    "title": "",
    "key": "row",
    "width": 80,
    "align": "center",
    "fixed": TableV2FixedDir.LEFT,
    cellRenderer: ({ rowIndex }: any) => h(
      'span',
      `${rowIndex + 1}`
    ),
  },
  {
    "title": "Repo Name",
    "key": "full_name",
    "dataKey": "full_name",
    "width": 400,
    "sortable": true,
    cellRenderer: ({ cellData, rowData }) => h(
      ElLink,
      {
        href: `https://${rowData['github_host']}/${cellData}`,
        target: "_blank",
        underline: false,
        onClick: (e) => e.stopPropagation(),
      },
      () => cellData
    )
  }
]

const filterCCs: ColumnConfig[] = []
const fetchColumns = () => {
  useFetch("/api/v1/columns", {
    method: "GET",
    onResponse({ response }) {
      response._data.forEach((cc: ColumnConfig) => {
        if (cc.filter) {
          filterCCs.push(cc)
        }
        if (cc.show) {
          let c: Column<any> = {
            "title": cc.title,
            "key": cc.key,
            "dataKey": cc.key,
            "width": cc.width,
            "sortable": true
          }
          if (cc.type != "string") {
            c["align"] = "center"
          }
          if (cc.type == "boolean") {
            c["cellRenderer"] = ({ cellData }) => h(
              Icon,
              {
                name: cellData ? "i-fa6-solid-check" : "i-fa6-solid-xmark",
                style: cellData ? "color: green" : "color: red"
              }
            )
          }
          if (cc.description) {
            const d = cc.description
            const t = cc.title
            c["headerCellRenderer"] = () => {
              return h(
                'div',
                {
                  class: "el-table-v2__header-cell-text",
                  title: d
                },
                h('span', t)
              )
            }
          }
          repos_table.columns.push(c)
        }
      })

      fetchRepos()
    }
  })
}

const repos_table = reactive({
  selectedDeviceID: null,
  rowKey: "repo",
  columns: columns,
  data: [],
  selected: new Map(),
  "sortState": ref<SortBy>({
    key: "full_name",
    order: TableV2SortOrder.ASC
  }),
  "onSort": (sortBy: SortBy) => {
    repos_table.data = useOrderBy(
      repos_table.data,
      [sortBy.key],
      [sortBy.order],
    )
    repos_table.sortState = sortBy
  },
  "rowClass": ({ rowIndex }: Parameters<RowClassNameGetter<any>>[0]) => {
    if (rowIndex % 2 === 1) {
      return 'zebra'
    }
    return ''
  },
})

const actionAPI = (api: string) => {
  useFetch(api, {
    method: "POST",
    body: {
      ids: Array.from(repos_table.selected.keys())
    },
    onResponse: ({ response }) => {
      if (response.status == 200) {
        ElNotification({
          title: 'Success',
          message: 'Operation success',
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
    }
  })
}

const actions = [
  [
    {
      label: 'Add Default Branch Protection Rule',
      click: () => actionAPI("/api/v1/repos/action/add-branch-protection-rule")
    }
  ],
  [
    {
      label: 'Requires PR: enabled',
      click: () => actionAPI("/api/v1/repos/action/requires-pr")
    }
  ],
  [
    {
      label: 'Requires Approving Review Count: 2',
      click: () => actionAPI("/api/v1/repos/action/required-approving-review-count")
    }
  ],
  [
    {
      label: 'Dismiss Stale Review: enabled',
      click: () => actionAPI("/api/v1/repos/action/dismisses-stale-reviews")
    }
  ],
  [
    {
      label: 'Requires Conversation Resolution: enabled',
      click: () => actionAPI("/api/v1/repos/action/requires-conversation-resolution")
    }
  ],
  [
    {
      label: 'Allow Force Pushes: disabled',
      click: () => actionAPI("/api/v1/repos/action/allows-force-pushes")
    }
  ],
  [
    {
      label: 'Allow Deletions: disabled',
      click: () => actionAPI("/api/v1/repos/action/allows-deletions")
    }
  ],
]

onMounted(() => {
  fetchColumns()
})
</script>

<template>
  <div class="common-layout">
    <el-container>
      <el-aside width="300px">
        <div class="filter-buttons">
          <el-button @click="exportToCSV"
                     circle
                     size="large"
                     class="filter-button">
            <UIcon name="i-fa6-solid-file-csv" />
          </el-button>
          <el-badge :value="repos_table.data.length"
                    :max="1000000"
                    class="filter-button"
                    type="primary">
            <el-button @click="refreshFilter"
                       size="large"
                       circle>
              <UIcon name="i-fa6-solid-arrows-rotate" />
            </el-button>
          </el-badge>
          <UDropdown :items="actions"
                     mode="click"
                     :popper="{ placement: 'bottom-start' }"
                     v-if="repos_table.selected.size > 0"
                     class="actions-button">
            <UButton color="white"
                     label="Actions"
                     trailing-icon="i-heroicons-chevron-down-20-solid" />
            <template #item="{ item }">
              <span>{{ item.label }}</span>
            </template>
          </UDropdown>
        </div>
        <Filter v-if="filterCCs.length > 0"
                v-for="c in filterCCs"
                :type="c.type"
                :title="c.title"
                :field="c.key"
                :expand="c.filter_expanded"
                :filters="filters"
                :filtersOrder="filtersOrder"
                @updateFilters="updateFilters"
                :disabled="loading" />
      </el-aside>
      <el-main>
        <div :style="{ height: 'calc(100vh - 150px)' }">
          <el-auto-resizer>
            <template #default="{ height, width }">
              <el-table-v2 :row-key="repos_table.rowKey"
                           :columns="repos_table.columns"
                           :data="repos_table.data"
                           :width="width"
                           :height="height"
                           fixed
                           :sort-by="repos_table.sortState"
                           @column-sort="repos_table.onSort"
                           :row-class="repos_table.rowClass">
                <template #overlay
                          v-if="loading">
                  <div class="el-loading-mask"
                       style="display: flex; align-items: center; justify-content: center">
                    <el-icon class="is-loading"
                             color="var(--el-color-primary)"
                             :size="26">
                      <loading-icon />
                    </el-icon>
                  </div>
                </template>
              </el-table-v2>
            </template>
          </el-auto-resizer>
        </div>
      </el-main>
    </el-container>
  </div>
</template>

<style scoped>
.example-showcase .el-table-v2__overlay {
  z-index: 9;
}

.el-aside {
  direction: rtl;
  height: calc(100vh - 125px);
  padding: 20px;
}

.el-collapse {
  direction: ltr;
}

.filter-button {
  margin-bottom: 9px;
  margin-right: 8px;
}

.filter-buttons {
  direction: ltr;
}

.actions-button {
  float: right;
  margin-top: 4px;
}
</style>
