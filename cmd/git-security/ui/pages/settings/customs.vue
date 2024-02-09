<script setup lang="ts">
type KeyValue = {
  key: string
  value: string
}
type CustomType = 'string' | 'number' | 'boolean'
type CustomConfig = {
  id: string
  pattern: string
  image: string
  command: string
  envs: KeyValue[]
  value_type: CustomType
  field: string
  default_value: any
  error_value: any
  enabled: boolean
}

const customs = ref<CustomConfig[]>([])
const fetchCustoms = () => {
  useFetch("/api/v1/customs", {
    method: "GET",
    onResponse({ response }) {
      customs.value.splice(0)
      response._data.forEach((cc: CustomConfig) => {
        customs.value.push(cc)
      })
    }
  })
}

const cast = (value: any, toType: string) => {
  if (typeof value === 'number' && isFinite(value)) {
    if (toType == "number") {
      return value
    } else if (toType == "string") {
      return String(value)
    } else {
      return value > 0 ? true : false
    }
  } else if (typeof value === 'boolean') {
    if (toType == "boolean") {
      return value
    } else if (toType == "string") {
      return String(value)
    } else {
      return value ? 1 : 0
    }
  } else {
    if (toType == "string") {
      return value
    } else if (toType == "number") {
      return isNaN(parseFloat(value)) ? 0 : parseFloat(value)
    } else {
      return value.length >= 0 && value.toLowerCase() != 'false' && value.toLowerCase() != '0'
    }
  }
}

const customChanged = (index: number) => {
  const c = customs.value[index]
  c.default_value = cast(c.default_value, c.value_type)
  c.error_value = cast(c.error_value, c.value_type)
  setTimeout(() => {
    useFetch(`/api/v1/custom/${c.id}`, {
      method: "PUT",
      body: c,
      onResponse({ response }) {
        if (response.status == 200) {
          ElNotification({
            title: 'Success',
            message: 'Custom config was updated successfully',
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
        fetchCustoms()
      }
    })
  }, 500)
}

const deleteCustom = (id: string) => {
  useFetch(`/api/v1/custom/${id}`, {
    method: "DELETE",
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: 'Success',
          message: 'Custom config was deleted successfully',
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
      fetchCustoms()
    }
  })
}

const addCustom = () => {
  useFetch("/api/v1/customs", {
    method: "POST",
    onResponse({ response }) {
      if (response.status == 200) {
        ElNotification({
          title: 'Success',
          message: 'New custom config was added successfully',
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
      fetchCustoms()
    }
  })
}

const addCustomEnv = (index: number) => {
  const c = customs.value[index]
  if (c.envs == undefined) {
    c.envs = []
  }
  c.envs.push({
    key: "",
    value: "",
  })
  customChanged(index)
}

const removeCustomEnv = (index: number, j: number) => {
  const c = customs.value[index]
  c.envs.splice(j, 1)
  customChanged(index)
}

onMounted(() => {
  fetchCustoms()
})
</script>

<template>
  <div class="add-button">
    <UButton icon="i-fa6-solid-plus"
             color="gray"
             variant="ghost"
             aria-label="Theme"
             @click="addCustom" />
  </div>
  <el-card shadow="always"
           class="custom"
           v-for="(element, index) in customs">
    <template #header>
      <div class="card-header">
        <span>#{{ index + 1 }} {{ element.field }}</span>
        <el-switch v-model="element.enabled"
                   class="enable-button"
                   @change="customChanged(index)" />
      </div>
    </template>
    <div>
      <el-input v-model="element.field"
                class="w-30 m-2"
                :placeholder="element.field"
                size="large"
                @change="customChanged(index)">
        <template #prepend>Field</template>
      </el-input>

      <el-input v-model="element.pattern"
                class="w-60 m-2"
                :placeholder="element.pattern"
                size="large"
                @change="customChanged(index)">
        <template #prepend>Repo Pattern</template>
      </el-input>
    </div>

    <div>
      <el-input v-model="element.image"
                class="w-60 m-2"
                placeholder="alpine"
                size="large"
                @change="customChanged(index)">
        <template #prepend>Container Image</template>
      </el-input>

      <el-input v-model="element.command"
                class="w-30 m-2"
                :placeholder="element.command"
                size="large"
                @change="customChanged(index)">
        <template #prepend>Command</template>
      </el-input>
    </div>

    <div>
      <el-card class="env-card"
               shadow="never">
        <template #header
                  style="{margin: '10px'}">
          <div class="env-card-header">
            <span>Environmental Variables</span>
            <UButton class="env-add-button"
                     icon="i-fa6-solid-plus"
                     color="gray"
                     variant="ghost"
                     aria-label="Theme"
                     @click="addCustomEnv(index)" />
          </div>
        </template>
        <div v-for="(env, j) in element.envs">
          <el-input v-model="env.key"
                    class="w-30 m-2"
                    size="large"
                    @change="customChanged(index)">
            <template #prepend>Key</template>
          </el-input>

          <el-input v-model="env.value"
                    class="w-60 m-2"
                    size="large"
                    :show-password="true"
                    @change="customChanged(index)">
            <template #prepend>Value</template>
          </el-input>

          <UButton class="env-delete-button"
                   icon="i-fa6-solid-xmark"
                   color="gray"
                   variant="ghost"
                   aria-label="Theme"
                   @click="removeCustomEnv(index, j)" />
        </div>
      </el-card>
    </div>

    <div>
      <el-select v-model="element.value_type"
                 class="w-30 m-2"
                 placeholder="Value Type"
                 size="large"
                 @change="customChanged(index)">
        <template #prefix>Value Type</template>
        <el-option key="string"
                   label="String"
                   value="string" />
        <el-option key="number"
                   label="Number"
                   value="number" />
        <el-option key="boolean"
                   label="Boolean"
                   value="boolean" />
      </el-select>
      <template v-if="element.value_type != 'boolean'">
        <el-input v-model="element.default_value"
                  class="w-30 m-2"
                  :type="element.value_type == 'string' ? 'text' : 'number'"
                  :placeholder="element.default_value ? element.default_value.toString() : ''"
                  size="large"
                  @change="customChanged(index)">
          <template #prepend>Default Value</template>
        </el-input>
        <el-input v-model="element.error_value"
                  class="w-30 m-2"
                  :type="element.value_type == 'string' ? 'text' : 'number'"
                  :placeholder="element.error_value ? element.error_value.toString() : ''"
                  size="large"
                  @change="customChanged(index)">
          <template #prepend>Error Value</template>
        </el-input>
      </template>
      <template v-if="element.value_type == 'boolean'">
        <el-checkbox v-model="element.default_value"
                     label="Default Value"
                     class="m-2"
                     size="large"
                     border
                     @change="customChanged(index)" />
        <el-checkbox v-model="element.error_value"
                     label="Error Value"
                     class="m-2"
                     size="large"
                     border
                     @change="customChanged(index)" />
      </template>
      <el-button type="danger"
                 class="delete-button"
                 circle
                 plain
                 @click="deleteCustom(element.id)">
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
  margin-top: 11px;
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

.custom {
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
