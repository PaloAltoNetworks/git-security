<script setup lang="ts">
type User = {
  name: string;
  roles: string[];
  adding: boolean;
};

const newRole = ref("");
const users = ref<User[]>([]);
const fetchUsers = () => {
  $fetch("/api/v1/users", {
    method: "GET",
    onResponse({ response }) {
      users.value = response._data;
    },
  });
};

const roles = ref<string[]>([]);
const fetchRoles = () => {
  $fetch("/api/v1/roles", {
    method: "GET",
    onResponse({ response }) {
      roles.value = response._data;
    },
  });
};

type Logged = {
  username: string;
  duration: number;
};
const loggeds = ref<Record<string, number>>({});
const fetchLogged = () => {
  $fetch("/api/v1/logged", {
    method: "GET",
    onResponse({ response }) {
      loggeds.value = {};
      response._data.forEach((logged: Logged) => {
        loggeds.value[logged.username] = logged.duration;
      });
    },
  });
};

const deleteTag = (idx: number, deletedRole: string) => {
  var roles = users.value[idx].roles.filter((e) => e !== deletedRole);
  updateUserRoleAPI(users.value[idx].name, roles);
};

const addRole = (idx: number) => {
  var roles = users.value[idx].roles;
  roles.push(newRole.value);
  updateUserRoleAPI(users.value[idx].name, roles);
};

const updateUserRoleAPI = (name: string, roles: string[]) => {
  $fetch("/api/v1/user/" + name, {
    method: "PUT",
    body: {
      roles: roles,
    },
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: "Success",
          message: "User roles were updated successfully",
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
      newRole.value = "";
      fetchUsers();
    },
  });
};

onMounted(() => {
  fetchUsers();
  fetchRoles();
  fetchLogged();
});
</script>

<template>
  <el-table :data="users" height="calc(100vh - 150px)" style="width: 100%">
    <el-table-column prop="name" label="Name" width="500" />
    <el-table-column label="Logged time in the last 30 days" width="200">
      <template #default="scope">
        <span v-if="loggeds[scope.row.name] > 0">{{
          useDayjs().duration(loggeds[scope.row.name], "seconds").humanize()
        }}</span>
      </template>
    </el-table-column>
    <el-table-column label="Roles">
      <template #default="scope">
        <div class="flex gap-2">
          <el-tag
            v-for="r in scope.row.roles"
            :key="r"
            :closable="scope.row.roles.length > 1"
            size="large"
            @close="deleteTag(scope.$index, r)"
          >
            {{ r }}
          </el-tag>

          <el-select
            v-model="newRole"
            placeholder="Select"
            style="width: 120px"
            v-if="scope.row.adding"
            @change="addRole(scope.$index)"
          >
            <el-option
              v-for="role in roles"
              :key="role"
              :label="role"
              :value="role"
            />
          </el-select>

          <el-button
            v-else
            class="button-new-tag"
            @click="scope.row.adding = true"
          >
            +
          </el-button>
        </div>
      </template>
    </el-table-column>
  </el-table>
</template>

<style scoped></style>
