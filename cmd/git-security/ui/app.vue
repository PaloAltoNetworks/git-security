<script setup lang="ts">
const route = useRoute();
var selected = ref("");
if (route.path == "/") {
  selected.value = "/repos";
} else {
  const arr = route.path.split("/", 2);
  selected.value = "/" + arr[1];
}
const handleSelect = (key: string) => {
  navigateTo({
    path: key,
  });
};

const colorMode = useColorMode();
const isDark = computed({
  get() {
    return colorMode.value === "dark";
  },
  set() {
    colorMode.preference = colorMode.value === "dark" ? "light" : "dark";
  },
});

const logout = () => {
  window.location.href = "/logout";
};
</script>

<template>
  <div class="top-right">
    <UButton
      :icon="isDark ? 'i-heroicons-moon-20-solid' : 'i-heroicons-sun-20-solid'"
      color="gray"
      variant="ghost"
      aria-label="Theme"
      @click="isDark = !isDark"
    />
    <UButton
      icon="i-fa6-solid-power-off"
      color="gray"
      variant="ghost"
      aria-label="Theme"
      @click="logout"
    />
  </div>
  <div>
    <el-image :src="isDark ? '/logo-white.png' : '/logo.png'" class="logo" />
    <el-menu
      :default-active="selected"
      class="el-menu-demo"
      mode="horizontal"
      @select="handleSelect"
    >
      <el-menu-item index="/repos">Repositories</el-menu-item>
      <el-menu-item index="/changelog">Changelog</el-menu-item>
      <el-menu-item index="/settings">Settings</el-menu-item>
    </el-menu>
    <NuxtPage />
  </div>
</template>

<style scoped>
.top-right {
  right: 20px;
  top: 28px;
  position: absolute;
  z-index: 2147483647;
}

.top-right-icon {
  margin-right: 10px;
}

.logo {
  width: 55px;
  float: left;
  margin-top: 3px;
  margin-right: 10px;
}
</style>
