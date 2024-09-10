<script setup lang="ts">
type KeyValue = {
  key: string;
  value: string;
};
type AutomationType = "string" | "number" | "boolean" | "array";
type AutomationConfig = {
  id: string;
  pattern: string;
  owner: string;
  exclude: string;
  image: string;
  command: string;
  envs: KeyValue[];
  enabled: boolean;
};

const automations = ref<AutomationConfig[]>([]);
const fetchAutomations = () => {
  $fetch("/api/v1/automations", {
    method: "GET",
    onResponse({ response }) {
      automations.value.splice(0);
      response._data.forEach((cc: AutomationConfig) => {
        automations.value.push(cc);
      });
    },
  });
};

const cast = (value: any, toType: string) => {
  if (typeof value === "number" && isFinite(value)) {
    if (toType == "number") {
      return value;
    } else if (toType == "string") {
      return String(value);
    } else if (toType == "boolean") {
      return value > 0 ? true : false;
    } else {
      return "";
    }
  } else if (typeof value === "boolean") {
    if (toType == "boolean") {
      return value;
    } else if (toType == "string") {
      return String(value);
    } else if (toType == "number") {
      return value ? 1 : 0;
    } else {
      return "";
    }
  } else if (typeof value === "object") {
    if (toType == "boolean") {
      return value.length > 0;
    } else if (toType == "string") {
      return "";
    } else if (toType == "number") {
      return value.length ? 1 : 0;
    } else {
      return "";
    }
  } else {
    if (toType == "string") {
      return value;
    } else if (toType == "number") {
      return isNaN(parseFloat(value)) ? 0 : parseFloat(value);
    } else if (toType == "string") {
      return (
        value.length >= 0 &&
        value.toLowerCase() != "false" &&
        value.toLowerCase() != "0"
      );
    } else {
      return "";
    }
  }
};

const automationChanged = (index: number) => {
  const c = automations.value[index];
  c.default_value = cast(c.default_value, c.value_type);
  c.error_value = cast(c.error_value, c.value_type);
  setTimeout(() => {
    $fetch(`/api/v1/automation/${c.id}`, {
      method: "PUT",
      body: c,
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: "Success",
            message: "Automation config was updated successfully",
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
        fetchAutomations();
      },
    });
  }, 500);
};

const deleteAutomation = (id: string) => {
  $fetch(`/api/v1/automation/${id}`, {
    method: "DELETE",
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: "Success",
          message: "Automation config was deleted successfully",
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
      fetchAutomations();
    },
  });
};

const addAutomation = () => {
  $fetch("/api/v1/automations", {
    method: "POST",
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: "Success",
          message: "New automation config was added successfully",
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
      fetchAutomations();
    },
  });
};

const addAutomationEnv = (index: number) => {
  const c = automations.value[index];
  if (c.envs == undefined) {
    c.envs = [];
  }
  c.envs.push({
    key: "",
    value: "",
  });
  automationChanged(index);
};

const removeAutomationEnv = (index: number, j: number) => {
  const c = automations.value[index];
  c.envs.splice(j, 1);
  automationChanged(index);
};

onMounted(() => {
  fetchAutomations();
});
</script>

<template>
  <div class="add-button">
    <UButton
      icon="i-fa6-solid-plus"
      color="gray"
      variant="ghost"
      aria-label="Theme"
      @click="addAutomation"
    />
  </div>
  <el-card
    shadow="always"
    class="automation"
    v-for="(element, index) in automations"
  >
    <template #header>
      <div class="card-header">
        <span>#{{ index + 1 }}</span>
        <el-switch
          v-model="element.enabled"
          class="enable-button"
          @change="automationChanged(index)"
        />
      </div>
    </template>

    <div>
      <el-input
        v-model="element.pattern"
        class="w-30 m-2"
        :placeholder="element.pattern"
        size="large"
        @change="automationChanged(index)"
      >
        <template #prepend>Repo Pattern</template>
      </el-input>

      <el-input
        v-model="element.owner"
        class="w-30 m-2"
        :placeholder="element.owner"
        size="large"
        @change="automationChanged(index)"
      >
        <template #prepend>Repo Owner</template>
      </el-input>

      <el-input
        v-model="element.exclude"
        class="w-30 m-2"
        :placeholder="element.exclude"
        size="large"
        @change="automationChanged(index)"
      >
        <template #prepend>Exclude Pattern</template>
      </el-input>
    </div>

    <div>
      <el-input
        v-model="element.image"
        class="w-60 m-2"
        placeholder="alpine"
        size="large"
        @change="automationChanged(index)"
      >
        <template #prepend>Container Image</template>
      </el-input>

      <el-input
        v-model="element.command"
        class="w-30 m-2"
        :placeholder="element.command"
        size="large"
        @change="automationChanged(index)"
      >
        <template #prepend>Command</template>
      </el-input>
    </div>

    <div>
      <el-card class="env-card" shadow="never">
        <template
          #header
          style="
             {
              margin: '10px';
            }
          "
        >
          <div class="env-card-header">
            <span>Environmental Variables</span>
            <UButton
              class="env-add-button"
              icon="i-fa6-solid-plus"
              color="gray"
              variant="ghost"
              aria-label="Theme"
              @click="addAutomationEnv(index)"
            />
          </div>
        </template>
        <div v-for="(env, j) in element.envs">
          <el-input
            v-model="env.key"
            class="w-30 m-2"
            size="large"
            @change="automationChanged(index)"
          >
            <template #prepend>Key</template>
          </el-input>

          <el-input
            v-model="env.value"
            class="w-60 m-2"
            size="large"
            :show-password="true"
            @change="automationChanged(index)"
          >
            <template #prepend>Value</template>
          </el-input>

          <UButton
            class="env-delete-button"
            icon="i-fa6-solid-xmark"
            color="gray"
            variant="ghost"
            aria-label="Theme"
            @click="removeAutomationEnv(index, j)"
          />
        </div>
      </el-card>
    </div>

    <div>
      <el-button
        type="danger"
        class="delete-button"
        circle
        plain
        @click="deleteAutomation(element.id)"
      >
        <UIcon name="i-fa6-solid-trash-can" />
      </el-button>
    </div>
  </el-card>
</template>

<style scoped>
.add-button {
  text-align: right;
  margin: 10px;
}

.delete-button {
  float: right;
  margin-bottom: 10px;
}

.enable-button {
  float: right;
}

.env-add-button {
  float: right;
  margin-top: -5px;
  margin-right: -10px;
}

.env-delete-button {
  vertical-align: middle;
}

.automation {
  margin-bottom: 10px;
}

.env-card {
  margin-left: 8px;
  margin-top: 8px;
  margin-bottom: 8px;
  width: 92%;
}

.handle {
  float: right;
  margin-top: 4px;
  margin-right: 10px;
  cursor: pointer;
}

.w-20 {
  width: 20%;
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
  font-weight: bold;
}

.env-card-header {
  margin-top: 10px;
  font-size: 14px;
}

.env-card :deep(.el-card__header) {
  background-color: var(--el-fill-color-light);
  color: var(--el-color-info);
  padding-top: 1px;
  padding-bottom: 9px;
  padding-left: 18px;
}

.env-card :deep(.el-card__body) {
  padding: 0;
}
</style>
