<script setup lang="ts">
import { showConfirmationDialog } from "@/common-functions";

type ColumnType = "string" | "number" | "boolean" | "array" | "date";
type ColumnConfig = {
  id: string;
  type: ColumnType;
  title: string;
  description: string;
  key: string;
  width: number;
  show: boolean;
  filter: boolean;
  filter_expanded?: boolean;
  csv: boolean;
  order: string;
  group_by: string;
};

const collapsed = ref<Record<string, boolean>>({});

const columns = ref<ColumnConfig[]>([]);
const fetchColumns = () => {
  $fetch("/api/v1/columns", {
    method: "GET",
    onResponse({ response }) {
      columns.value.splice(0);
      response._data.forEach((cc: ColumnConfig) => {
        columns.value.push(cc);
        const storedState = sessionStorage.getItem(cc.id);
        collapsed.value[cc.id] = storedState ? JSON.parse(storedState) : true;
      });
    },
  });
};

const moved = (e: any) => {
  if (e.oldIndex != e.newIndex && columns.value.length > 1) {
    let currentIndex = e.newIndex - 1;
    const movedColumn: ColumnConfig = columns.value[currentIndex];
    let prev = "";
    let next = "";
    if (currentIndex == 0) {
      next = columns.value[1].order;
    } else if (currentIndex == columns.value.length - 1) {
      prev = columns.value[currentIndex - 1].order;
    } else {
      prev = columns.value[currentIndex - 1].order;
      next = columns.value[currentIndex + 1].order;
    }
    $fetch("/api/v1/columns/order", {
      method: "POST",
      body: {
        id: movedColumn.id,
        prev: prev,
        next: next,
      },
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: "Success",
            message: "Order was changed successfully",
            type: "success",
            position: "bottom-right",
          });
        } else {
          ElNotification({
            title: "Error",
            message: "Internal error occurred",
            type: "error",
            position: "bottom-right",
          });
        }
        fetchColumns();
      },
    });
  }
};

const columnChanged = (index: number) => {
  setTimeout(() => {
    const c = columns.value[index];
    $fetch(`/api/v1/column/${c.id}`, {
      method: "PUT",
      body: c,
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: "Success",
            message: "Column config was updated successfully",
            type: "success",
            position: "bottom-right",
          });
        } else {
          ElNotification({
            title: "Error",
            message: "Internal error occurred",
            type: "error",
            position: "bottom-right",
          });
        }
        fetchColumns();
      },
    });
  }, 500);
};

const deleteColumn = async (id: string, column: string) => {
  try {
    const confirmed = await showConfirmationDialog(
      `Are you sure you want to delete the column: \n${column} ?`
    );
    if (confirmed) {
      $fetch(`/api/v1/column/${id}`, {
        method: "DELETE",
        onResponse({ response }) {
          if (response.status == 200) {
            ElNotification({
              title: "Success",
              message: "Column config was deleted successfully",
              type: "success",
              position: "bottom-right",
            });
          } else {
            ElNotification({
              title: "Error",
              message: "Internal error occurred",
              type: "error",
              position: "bottom-right",
            });
          }
          fetchColumns();
        },
      });
    }
  } catch (error) {
    console.error(error);
  }
};

const addColumn = () => {
  $fetch("/api/v1/columns", {
    method: "POST",
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: "Success",
          message: "New column config was added successfully",
          type: "success",
          position: "bottom-right",
        });
      } else {
        ElNotification({
          title: "Error",
          message: "Internal error occurred",
          type: "error",
          position: "bottom-right",
        });
      }
      fetchColumns();
    },
  });
};

const toggleCollapse = (id: string) => {
  collapsed.value[id] = !collapsed.value[id];
  sessionStorage.setItem(id, JSON.stringify(collapsed.value[id]));
};

onMounted(() => {
  fetchColumns();
});
</script>

<template>
  <draggable
    tag="div"
    :list="columns"
    handle=".handle"
    item-key="id"
    @end="moved"
  >
    <template #header>
      <div class="add-button">
        <UButton
          icon="i-fa6-solid-plus"
          color="gray"
          variant="ghost"
          aria-label="Theme"
          @click="addColumn"
        />
      </div>
    </template>
    <template #item="{ element, index }">
      <el-card shadow="always" class="column">
        <template #header>
          <div class="card-header">
            <div>
              <span class="title">#{{ index + 1 }} {{ element.title }}</span>
              <el-checkbox
                v-model="element.show"
                label="Show in table?"
                class="m-2 m-3"
                size="large"
                border
                @change="columnChanged(index)"
              />
              <el-checkbox
                v-model="element.filter"
                label="Show in filters?"
                class="m-2"
                size="large"
                border
                v-if="element.type != 'date'"
                @change="columnChanged(index)"
              />
              <el-checkbox
                v-model="element.filter_expanded"
                label="Expanded in filters?"
                class="m-2"
                size="large"
                border
                v-if="element.type != 'date'"
                @change="columnChanged(index)"
              />
              <el-button
                type="danger"
                class="delete-button"
                circle
                plain
                @click="deleteColumn(element.id, element.title)"
              >
                <UIcon name="i-fa6-solid-trash-can" />
              </el-button>
            </div>
            <div>
              <UIcon
                :name="
                  collapsed[element.id]
                    ? 'i-fa6-solid-chevron-down'
                    : 'i-fa6-solid-chevron-up'
                "
                class="handle"
                @click="toggleCollapse(element.id)"
              />
              <UIcon name="i-fa6-solid-align-justify" class="handle" />
            </div>
          </div>
        </template>
        <div v-show="!collapsed[element.id]">
          <el-input
            v-model="element.title"
            class="w-30 m-2"
            :placeholder="element.title"
            size="large"
            @change="columnChanged(index)"
          >
            <template #prepend>Title</template>
          </el-input>
          <el-autocomplete
            v-model="element.key"
            :fetch-suggestions="columnDataKeys"
            class="m-2"
            size="large"
            @change="columnChanged(index)"
            :style="{ width: '30%' }"
          >
            <template #prepend>Data Key</template>

            <template #default="{ item }">
              <div>{{ item.displayValue }}</div>
            </template>
          </el-autocomplete>
          <el-input
            v-model.number="element.width"
            type="number"
            class="w-30 m-2"
            size="large"
            @change="columnChanged(index)"
          >
            <template #prepend>Column Width</template>
          </el-input>
        </div>
        <div v-show="!collapsed[element.id]">
          <el-select
            v-model="element.type"
            class="w-30 m-2"
            placeholder="Column Type"
            size="large"
            @change="columnChanged(index)"
          >
            <template #prefix>Data Type</template>
            <el-option key="string" label="String" value="string" />
            <el-option key="number" label="Number" value="number" />
            <el-option key="boolean" label="Boolean" value="boolean" />
            <el-option key="array" label="Array" value="array" />
            <el-option key="date" label="Date" value="date" />
          </el-select>
          <el-input
            v-model="element.description"
            class="w-60 m-2"
            :placeholder="element.description"
            size="large"
            @change="columnChanged(index)"
          >
            <template #prepend>Description</template>
          </el-input>
        </div>
        <div v-show="!collapsed[element.id]">
          <el-select
            v-model="element.group_by"
            class="w-30 m-2"
            placeholder="Calculated Field"
            size="large"
            @change="columnChanged(index)"
          >
            <template #prefix>Group By</template>
            <el-option key="none" label="None" value="" />
            <el-option key="sum" label="Sum" value="sum" />
            <el-option key="avg" label="Avg" value="avg" />
            <el-option key="min" label="Min" value="min" />
            <el-option key="max" label="Max" value="max" />
            <el-option
              key="distinct_count_by_count"
              label="Distinct Count"
              value="distinct_count_by_count"
            />
            <el-option
              key="distinct_count_by_name_asc"
              label="Distinct Count By Name (ASC)"
              value="distinct_count_by_name_asc"
            />
            <el-option
              key="distinct_count_by_name_desc"
              label="Distinct Count By Name (DESC)"
              value="distinct_count_by_name_desc"
            />
          </el-select>
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
