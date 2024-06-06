<script setup lang="ts">
import { ElLink, TableV2FixedDir, TableV2SortOrder } from "element-plus";
import type { Column, RowClassNameGetter, SortBy } from "element-plus";
import { Loading as LoadingIcon } from "@element-plus/icons-vue";
import Icon from "#ui/components/elements/Icon.vue";
import HeaderCellRenderer from "element-plus/es/components/table-v2/src/renderers/header-cell.mjs";

const loading = ref(false);
const filters = ref<Record<string, string[]>>({});
const negates = ref<Record<string, boolean>>({});
const filtersOrder = ref([]);
const updateFilters = (field: string) => {
  fetchChangelog();
};

const dateRange = reactive({ start: -14, end: 0 });
const updateDateRange = (start: number, end: number) => {
  dateRange.start = start;
  dateRange.end = end;
  fetchChangelog();
};
const marks = ref({
  0: "Now",
  "-14": "14 days ago",
});

const transform = (f: Record<string, string[]>) => {
  let results = [];
  for (let filter in f) {
    if (f[filter].length > 0) {
      results.push({
        field: filter,
        values: f[filter],
        negate: negates.value[filter] || false,
      });
    }
  }
  return results;
};

const fetchChangelog = () => {
  loading.value = true;
  $fetch("/api/v1/changelog", {
    method: "POST",
    body: {
      filters: transform(filters.value),
      start_date: getTimestamp(dateRange.start),
      end_date: getTimestamp(dateRange.end),
    },
    onResponse({ response }) {
      loading.value = false;
      changelog_table.data = response._data;
      changelog_table.data = useOrderBy(
        changelog_table.data,
        ["created_at"],
        [TableV2SortOrder.DESC]
      );
    },
  });
};

const refreshFilter = async () => {
  Object.keys(filters.value).forEach((key) => delete filters.value[key]);
  filtersOrder.value.splice(0);
  fetchChangelog();
};

const exportToCSV = async () => {
  $fetch("/api/v1/changelog?csv=true", {
    method: "POST",
    body: {
      filters: transform(filters.value),
      start_date: getTimestamp(dateRange.start),
      end_date: getTimestamp(dateRange.end),
    },
    onResponse({ response }) {
      const url =
        "data:text/csv;charset=utf-8," + encodeURIComponent(response._data);
      const a = document.createElement("a");
      a.href = url;
      a.setAttribute("download", "repos_changelog.csv");
      a.click();
    },
  });
};

const getCellRenderer = ({ cellData }: { cellData: any }) => {
  let cd = cellData.toLowerCase();
  if (cd.startsWith("rgba")) {
    return h("div", {
      style: {
        "padding-top": "5px",
        "background-color": cellData,
        "border-radius": "50%",
        width: "31px",
        height: "31px",
      },
    });
  } else if (cd == "false" || cd == "true") {
    return h(Icon, {
      name: cd == "true" ? "i-fa6-solid-check" : "i-fa6-solid-xmark",
      style: cd == "true" ? "color: green" : "color: red",
    });
  }
  return cellData;
};

const getColumns = (): Column<any>[] => [
  {
    title: "",
    key: "row",
    width: 80,
    align: "center",
    fixed: TableV2FixedDir.LEFT,
    cellRenderer: ({ rowIndex }: any) => h("span", `${rowIndex + 1}`),
  },
  {
    title: "Repo Name",
    key: "full_name",
    dataKey: "full_name",
    width: 400,
    sortable: true,
    fixed: TableV2FixedDir.LEFT,
    cellRenderer: ({ cellData, rowData }) =>
      h(
        "div",
        {
          style: { padding: "5px 0" },
        },
        h(
          ElLink,
          {
            href: `https://${rowData["github_host"]}/${cellData}`,
            target: "_blank",
            underline: false,
            onClick: (e) => e.stopPropagation(),
          },
          () => cellData
        )
      ),
  },
  {
    title: "Repo Owner",
    key: "repo_owner",
    dataKey: "repo_owner",
    width: 200,
    sortable: true,
  },
  {
    title: "Field",
    key: "field",
    dataKey: "field",
    width: 300,
    sortable: true,
  },
  {
    title: "From",
    key: "from",
    dataKey: "from",
    width: 200,
    sortable: true,
    cellRenderer: getCellRenderer,
  },
  {
    title: "To",
    key: "to",
    dataKey: "to",
    width: 200,
    sortable: true,
    cellRenderer: getCellRenderer,
  },
  {
    title: "Timestamp",
    key: "created_at",
    dataKey: "created_at",
    width: 250,
    sortable: true,
  },
];

const changelog_table = reactive({
  rowKey: "id",
  columns: getColumns(),
  data: <any>[],
  sortState: ref<SortBy>({
    key: "created_at",
    order: TableV2SortOrder.DESC,
  }),
  onSort: (sortBy: SortBy) => {
    changelog_table.data = useOrderBy(
      changelog_table.data,
      [sortBy.key],
      [sortBy.order]
    );
    changelog_table.sortState = sortBy;
  },
  rowClass: ({ rowIndex }: Parameters<RowClassNameGetter<any>>[0]) => {
    if (rowIndex % 2 === 1) {
      return "zebra";
    }
    return "";
  },
});

onMounted(() => {
  fetchChangelog();
});
</script>

<template>
  <div class="common-layout">
    <el-container>
      <el-aside width="330px">
        <div class="filter-buttons">
          <el-button
            @click="exportToCSV"
            circle
            size="large"
            class="filter-button"
          >
            <UIcon name="i-fa6-solid-file-csv" />
          </el-button>
          <el-badge
            :value="changelog_table.data.length"
            :max="1000000"
            class="filter-button"
            type="primary"
          >
            <el-button @click="refreshFilter" size="large" circle>
              <UIcon name="i-fa6-solid-arrows-rotate" />
            </el-button>
          </el-badge>
          <DateSlider
            width="150px"
            :marks="marks"
            :default="[dateRange.start, dateRange.end]"
            :range="[-14, 0]"
            @updateDateRange="updateDateRange"
            :disabled="loading"
          />
        </div>
        <div class="filters">
          <ChangelogFilter
            type="string"
            title="Organization"
            field="owner.login"
            :expand="true"
            :filters="filters"
            :negates="negates"
            :filtersOrder="filtersOrder"
            @updateFilters="updateFilters"
            :dateRange="dateRange"
            :disabled="loading"
          />
          <ChangelogFilter
            type="string"
            title="Repo Name"
            field="name"
            :expand="true"
            :filters="filters"
            :negates="negates"
            :filtersOrder="filtersOrder"
            @updateFilters="updateFilters"
            :dateRange="dateRange"
            :disabled="loading"
          />
          <ChangelogFilter
            type="string"
            title="Repo Owner"
            field="repo_owner"
            :expand="false"
            :filters="filters"
            :negates="negates"
            :filtersOrder="filtersOrder"
            @updateFilters="updateFilters"
            :dateRange="dateRange"
            :disabled="loading"
          />
          <ChangelogFilter
            type="string"
            title="Field"
            field="field"
            :expand="false"
            :filters="filters"
            :negates="negates"
            :filtersOrder="filtersOrder"
            @updateFilters="updateFilters"
            :dateRange="dateRange"
            :disabled="loading"
          />
          <ChangelogFilter
            type="string"
            title="From"
            field="from"
            :expand="false"
            :filters="filters"
            :negates="negates"
            :filtersOrder="filtersOrder"
            @updateFilters="updateFilters"
            :dateRange="dateRange"
            :disabled="loading"
          />
          <ChangelogFilter
            type="string"
            title="To"
            field="to"
            :expand="false"
            :filters="filters"
            :negates="negates"
            :filtersOrder="filtersOrder"
            @updateFilters="updateFilters"
            :dateRange="dateRange"
            :disabled="loading"
          />
        </div>
      </el-aside>
      <el-main>
        <div :style="{ height: 'calc(100vh - 150px)' }">
          <el-auto-resizer>
            <template #default="{ height, width }">
              <el-table-v2
                :row-key="changelog_table.rowKey"
                :columns="changelog_table.columns"
                :data="changelog_table.data"
                :width="width"
                :height="height"
                fixed
                :sort-by="changelog_table.sortState"
                @column-sort="changelog_table.onSort"
                :row-class="changelog_table.rowClass"
              >
                <template #overlay v-if="loading">
                  <div
                    class="el-loading-mask"
                    style="
                      display: flex;
                      align-items: center;
                      justify-content: center;
                    "
                  >
                    <el-icon
                      class="is-loading"
                      color="var(--el-color-primary)"
                      :size="26"
                    >
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
  padding: 20px 20px 20px 0;
}

.filters {
  overflow: auto;
  height: calc(100vh - 200px);
  padding-left: 20px;
}

.el-collapse {
  direction: ltr;
}

.filter-button {
  margin-bottom: 9px;
  margin-right: 8px;
  margin-left: 0px;
}

.filter-buttons {
  direction: ltr;
  padding-left: 20px;
}
</style>
