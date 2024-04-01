<script setup lang="ts">
const route = useRoute();
const active = ref(route.path);

const select = (index: string) => {
  navigateTo(index);
};
</script>

<template>
  <div class="common-layout">
    <el-container>
      <el-aside width="200px">
        <el-menu
          :default-active="active"
          :style="{ height: 'calc(100vh - 150px)' }"
          @select="select"
        >
          <el-menu-item index="/settings">
            <UIcon name="i-fa6-solid-gear" class="menu-icon" />
            <span>General</span>
          </el-menu-item>
          <el-menu-item index="/settings/columns">
            <UIcon name="i-fa6-solid-table-columns" class="menu-icon" />
            <span>Columns</span>
          </el-menu-item>
          <el-menu-item index="/settings/customs">
            <UIcon name="i-fa6-solid-terminal" class="menu-icon" />
            <span>Custom Hooks</span>
          </el-menu-item>
          <el-menu-item index="/settings/score">
            <UIcon name="i-fa6-solid-calculator" class="menu-icon" />
            <span>Repository Score</span>
          </el-menu-item>
          <el-menu-item index="/settings/roles">
            <UIcon name="i-fa6-solid-person-circle-check" class="menu-icon" />
            <span>User Roles</span>
          </el-menu-item>
        </el-menu>
      </el-aside>
      <el-main>
        <div :style="{ height: 'calc(100vh - 150px)' }">
          <NuxtPage />
        </div>
      </el-main>
    </el-container>
  </div>
</template>

<style scoped>
.menu-icon {
  margin-right: 5px;
}
</style>
