<script setup lang="ts">
import { showConfirmationDialog } from "@/common-functions";

type Owner = {
  id: string;
  name: string;
  contact: string;
  notes: string;
};

const editingID = ref(undefined);
const owners = ref<Owner[]>([]);
const fetchOwners = () => {
  editingID.value = undefined;
  $fetch("/api/v1/owners", {
    method: "GET",
    onResponse({ response }) {
      owners.value = response._data;
    },
  });
};

const add = () => {
  $fetch("/api/v1/owners", {
    method: "POST",
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: "Success",
          message: "New custom config was added successfully",
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
      fetchOwners();
    },
  });
};

const update = (row: any) => {
  $fetch(`/api/v1/owner/${row.id}`, {
    method: "PUT",
    body: row,
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: "Success",
          message: "Owner was updated successfully",
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
      fetchOwners();
    },
  });
};

const edit = (row: any) => {
  editingID.value = row.id;
};

const cancel = () => {
  editingID.value = undefined;
  fetchOwners();
};

const deleteRow = async (row: any) => {
  const confirmed = await showConfirmationDialog(
    `Are you sure you want to delete the row ?`
  );
  if (confirmed) {
    $fetch(`/api/v1/owner/${row.id}`, {
      method: "DELETE",
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: "Success",
            message: "Owner was deleted successfully",
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
        fetchOwners();
      },
    });
  }
};

onMounted(() => {
  fetchOwners();
});
</script>

<template>
  <el-table :data="owners" height="calc(100vh - 150px)" style="width: 100%">
    <el-table-column prop="name" label="Name" width="250">
      <template #default="scope">
        <template v-if="editingID == scope.row.id">
          <el-input v-model="scope.row[scope.column.property]" />
        </template>
        <template v-else>
          {{ scope.row[scope.column.property] }}
        </template>
      </template>
    </el-table-column>
    <el-table-column prop="contact" label="Contact" width="500">
      <template #default="scope">
        <template v-if="editingID == scope.row.id">
          <el-input v-model="scope.row[scope.column.property]" />
        </template>
        <template v-else>
          {{ scope.row[scope.column.property] }}
        </template>
      </template>
    </el-table-column>
    <el-table-column prop="notes" label="Notes">
      <template #default="scope">
        <template v-if="editingID == scope.row.id">
          <el-input
            type="textarea"
            v-model="scope.row[scope.column.property]"
          />
        </template>
        <template v-else>
          <pre>{{ scope.row[scope.column.property] }}</pre>
        </template>
      </template>
    </el-table-column>
    <el-table-column fixed="right" label="Operations" width="150">
      <template #header>
        <el-button link type="primary" size="small" @click="add">
          Add
        </el-button>
      </template>
      <template #default="scope">
        <template v-if="editingID == scope.row.id">
          <el-button
            link
            type="primary"
            size="small"
            @click="update(scope.row)"
          >
            Update
          </el-button>
        </template>
        <template v-if="!editingID">
          <el-button link type="primary" size="small" @click="edit(scope.row)">
            Edit
          </el-button>
        </template>

        <template v-if="editingID == scope.row.id">
          <el-button link type="primary" size="small" @click="cancel">
            Cancel
          </el-button>
        </template>
        <template v-if="!editingID">
          <el-button
            link
            type="primary"
            size="small"
            @click="deleteRow(scope.row)"
          >
            Delete
          </el-button>
        </template>
      </template>
    </el-table-column>
  </el-table>
</template>

<style scoped></style>
