<script setup lang="ts">
import { showConfirmationDialog } from "@/common-functions";
import { columnDataKeys } from "@/utils/common-functions";

type GlobalSettings = {
  score_colors: Array<ScoreColor>;
  score_weights: Array<ScoreWeight>;
};

type ScoreColor = {
  label: string;
  range: Array<number>;
  color: string;
};

type ScoreWeight = {
  weight: number;
  field: string;
  comparator: string;
  arg: string;
};

const gs = ref<GlobalSettings>({
  score_colors: [],
  score_weights: [],
});
const fetchGlobalSettings = () => {
  $fetch("/api/v1/globalsettings", {
    method: "GET",
    onResponse({ response }) {
      gs.value = response._data;
    },
  });
};

const addColor = () => {
  gs.value.score_colors.push({
    label: "",
    range: [0, 0],
    color: "rgba(0, 0, 0, 1)",
  });
};

const addWeight = () => {
  gs.value.score_weights.push({
    weight: 0,
    field: "",
    comparator: "==",
    arg: "",
  });
};

const removeColor = async (index: number) => {
  const confirmed = await showConfirmationDialog(
    `Are you sure you want to delete the color ?`
  );
  if (confirmed) {
    gs.value.score_colors.splice(index, 1);
    globalSettingsChanged();
  }
};

const removeWeight = async (index: number) => {
  const confirmed = await showConfirmationDialog(
    `Are you sure you want to delete the weight ?`
  );
  if (confirmed) {
    gs.value.score_weights.splice(index, 1);
    globalSettingsChanged();
  }
};

const globalSettingsChanged = () => {
  setTimeout(() => {
    $fetch(`/api/v1/globalsettings`, {
      method: "PUT",
      body: gs.value,
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: "Success",
            message: "Global settings was updated successfully",
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
        fetchGlobalSettings();
      },
    });
  }, 500);
};

const predefineColors = ref([
  "#ff4500",
  "#ff8c00",
  "#ffd700",
  "#90ee90",
  "#00ced1",
  "#1e90ff",
  "#c71585",
  "rgba(255, 69, 0, 0.68)",
  "rgb(255, 120, 0)",
  "hsv(51, 100, 98)",
  "hsva(120, 40, 94, 0.5)",
  "hsl(181, 100%, 37%)",
  "hsla(209, 100%, 56%, 0.73)",
  "#c7158577",
]);

onMounted(() => {
  fetchGlobalSettings();
});
</script>

<template>
  <el-card
    shadow="always"
    :body-style="{ height: '400px', overflow: 'scroll' }"
  >
    <template #header>
      <div class="card-header">
        <span>Repository Score Colors</span>
        <div class="add-button">
          <UButton
            icon="i-fa6-solid-plus"
            color="gray"
            variant="ghost"
            aria-label="Theme"
            @click="addColor"
          />
        </div>
      </div>
    </template>
    <div v-for="(sc, index) in gs.score_colors">
      <span class="label">Label:</span>
      <el-input
        v-model="sc.label"
        class="w-20 m-2"
        size="large"
        @change="globalSettingsChanged()"
      >
      </el-input>
      <span class="label">Color:</span>
      <el-color-picker
        v-model="sc.color"
        show-alpha
        :predefine="predefineColors"
        @change="globalSettingsChanged()"
      />
      <span class="label"
        >Score range: ({{ sc.range[0] }} - {{ sc.range[1] }})</span
      >
      <el-slider
        class="slider w-20 m-2"
        v-model="sc.range"
        range
        show-stops
        :step="10"
        :max="100"
        @change="globalSettingsChanged()"
      />

      <UButton
        class="env-delete-button"
        icon="i-fa6-solid-xmark"
        color="gray"
        variant="ghost"
        aria-label="Theme"
        @click="removeColor(index)"
      />
    </div>
  </el-card>

  <el-card
    shadow="always"
    :body-style="{ height: '400px', overflow: 'scroll' }"
  >
    <template #header>
      <div class="card-header">
        <span>Repository Score Weights</span>
        <div class="add-button">
          <UButton
            icon="i-fa6-solid-plus"
            color="gray"
            variant="ghost"
            aria-label="Theme"
            @click="addWeight"
          />
        </div>
      </div>
    </template>
    <div v-for="(sw, index) in gs.score_weights">
      <span class="label">Weight: ({{ sw.weight }})</span>
      <el-slider
        class="slider w-20 m-2"
        v-model="sw.weight"
        :max="100"
        @change="globalSettingsChanged()"
      />

      <el-autocomplete
        v-model="sw.field"
        :fetch-suggestions="columnDataKeys"
        class="m-2"
        size="large"
        @change="globalSettingsChanged()"
        :style="{ width: '40%' }"
      >
        <template #prepend>Field</template>

        <template #default="{ item }">
          <div>{{ item.displayValue }}</div>
        </template>
      </el-autocomplete>

      <el-select
        v-model="sw.comparator"
        class="w-10 m-2"
        placeholder="Comparator"
        size="large"
        @change="globalSettingsChanged()"
      >
        <template #prefix>Comparator</template>
        <el-option key="==" label="==" value="==" />
        <el-option key=">=" label=">=" value=">=" />
        <el-option key="<=" label="<=" value="<=" />
        <el-option key=">" label=">" value=">" />
        <el-option key="<" label="<" value="<" />
        <el-option key="!=" label="!=" value="!=" />
      </el-select>

      <el-input
        v-model="sw.arg"
        class="w-10 m-2"
        size="large"
        @change="globalSettingsChanged()"
      >
        <template #prepend>Argument</template>
      </el-input>

      <UButton
        class="env-delete-button"
        icon="i-fa6-solid-xmark"
        color="gray"
        variant="ghost"
        aria-label="Theme"
        @click="removeWeight(index)"
      />
    </div>
  </el-card>
</template>

<style scoped>
.label {
  margin: 0px 20px;
}

.add-button {
  top: 0px;
  right: 0px;
  margin: 10px;
  float: right;
  position: relative;
}

.slider {
  max-width: 600px;
  display: inline-block;
  vertical-align: text-top;
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

.w-10 {
  width: 10%;
}

.w-20 {
  width: 20%;
}

.w-40 {
  width: 40%;
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

.m-3 {
  margin-left: 30px;
}
</style>
