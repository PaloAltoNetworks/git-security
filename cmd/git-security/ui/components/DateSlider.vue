<script setup lang="ts">
const props = defineProps({
  width: String,
  marks: Object,
  step: {
    type: Number,
    default: 1,
  },
  range: {
    type: Array<number>,
    required: true,
  },
  default: {
    type: Array<number>,
    required: true,
  },
  disabled: Boolean,
});

const value = ref(props.default);

const emit = defineEmits(["updateDateRange"]);

const changed = () => {
  emit("updateDateRange", value.value[0], value.value[1]);
};
</script>

<template>
  <div class="slider-block">
    <el-slider
      :style="'width:' + props.width"
      v-model="value"
      range
      :marks="props.marks"
      size="small"
      :step="props.step"
      :min="props.range[0]"
      :max="props.range[1]"
      :disabled="props.disabled"
      @change="changed"
    />
  </div>
</template>

<style scoped>
.slider-block {
  display: inline-flex;
  align-items: center;
  vertical-align: top;
}

.slider-block .el-slider {
  margin-top: 0;
  margin-left: 40px;
}
</style>
