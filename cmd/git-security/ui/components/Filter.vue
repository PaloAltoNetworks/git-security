<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'

type Filter = {
  field: string
  values: string[]
}

type Item = {
  name: any
  count: number
}

const props = defineProps({
  type: String,
  title: String,
  field: String,
  filters: Object,
  filtersOrder: {
    type: Array as PropType<Filter[]>,
    required: true,
  },
  expand: Boolean,
  disabled: Boolean,
})
const emit = defineEmits(['updateFilters'])

var checked = 0
const collapse = ref("")
const items = ref<Item[]>([])
const orderBy = ref(0) // 0: A-Z, 1: Z-A, 2, 9-1, 3: 1-9
const search = ref("")
const searchedItems = ref<Item[]>([])
const searchedItemsChecked = ref<Item[]>([])
const searchedItemsUnchecked = ref<Item[]>([])

watch([props.filters], () => {
  // set a timeout here to run the parent eventhandler first
  setTimeout(() => {
    fetchFilters()
  }, 0)
})

watch([items, search, props.filters], () => {
  searchedItems.value = []
  searchedItemsChecked.value = []
  searchedItemsUnchecked.value = []
  let checked = []
  if (props.filters![props.field!] != undefined) {
    checked = props.filters![props.field!].reduce((map: Record<string, boolean>, item_name: string) => {
      map[item_name] = true
      return map
    }, {})
  }
  for (let item of items.value) {
    if (search.value.trim() == '' || item.name.toLowerCase().indexOf(search.value.trim().toLowerCase()) > -1) {
      searchedItems.value.push(item)
      if (item.name in checked) {
        searchedItemsChecked.value.push(item)
      } else {
        searchedItemsUnchecked.value.push(item)
      }
    }
  }
})

const fetchFilters = async () => {
  let filters: Filter[] = []
  for (const elem of props.filtersOrder) {
    if (elem.field == props.field) {
      break
    } else {
      filters.push(elem)
    }
  }
  const { data } = useFetch(`/api/v1/repos/${props.field}`, {
    method: "POST",
    body: {
      filters: filters
    },
    onResponse({ response }) {
      items.value = response._data
      sortFilters()
    }
  })
}

const checkboxChanged = () => {
  let filtersOrder = props.filtersOrder
  let filters = props.filters
  let values = props.filters![props.field!]

  // search if it exists
  let index = -1
  for (const [i, elem] of filtersOrder.entries()) {
    if (elem.field == props.field) {
      index = i
      break
    }
  }

  if (values.length > 0) {
    if (index >= 0) {
      // need to detect whether it's removal
      if (checked > values.length) {
        // remove all the entries in filters
        for (let i = index + 1; i < filtersOrder.length; i++) {
          delete filters![filtersOrder[i].field]
        }
        // remove all the things behind index in filtersOrder
        filtersOrder.splice(index + 1)
      }
      filtersOrder[index] = {
        field: props.field!,
        values: values
      }
    } else {
      filtersOrder.push({
        field: props.field!,
        values: values
      })
    }
  } else {
    // it's total removal case
    for (let i = index; i < filtersOrder.length; i++) {
      delete filters![filtersOrder[i].field]
    }
    filtersOrder.splice(index)
  }
  checked = values.length
  emit('updateFilters', props.field)
}

const changeOrderAlphabet = () => {
  orderBy.value = orderBy.value < 2 ? 1 - orderBy.value : 0
  sortFilters()
}

const changeOrderCount = () => {
  orderBy.value = orderBy.value > 1 ? 5 - orderBy.value : 2
  sortFilters()
}

const sortFilters = () => {
  let key = orderBy.value > 1 ? "count" : "name"
  if (orderBy.value == 1 || orderBy.value == 2) {
    items.value = useOrderBy(
      items.value,
      [key],
      ["desc"])
  } else {
    items.value = useOrderBy(
      items.value,
      [key],
      ["asc"])
  }
}

const negateSelection = () => {
  if (props.filters![props.field!] != undefined) {
    for (let i = 0; i < searchedItemsChecked.value.length; i++) {
      const item = searchedItemsChecked.value[i]
      const index = props.filters![props.field!].indexOf(item.name)
      props.filters![props.field!].splice(index, 1)
    }
    props.filters![props.field!] = props.filters![props.field!].concat(searchedItemsUnchecked.value.map(item => item.name))
  }
  else {
    props.filters![props.field!] = searchedItems.value.map(item => item.name)
  }
  checkboxChanged()
}

const clearSelection = () => {
  for (let i = 0; i < searchedItemsChecked.value.length; i++) {
    const item = searchedItemsChecked.value[i]
    const index = props.filters![props.field!].indexOf(item.name)
    props.filters![props.field!].splice(index, 1)
  }
  checkboxChanged()
}

const selectAll = () => {
  if (props.filters![props.field!] == undefined) {
    props.filters![props.field!] = []
  }
  props.filters![props.field!] = props.filters![props.field!].concat(searchedItemsUnchecked.value.map(item => item.name))
  checkboxChanged()
}

const copyToClipboard = () => {
  let clipboardItems = '';
  for (let item of items.value) {
    clipboardItems += `${item.name} (${item.count})\n`;
  }
  navigator.clipboard.writeText(clipboardItems)
    .then(() => {
      alert('Filter data is copied to clipboard');
    })
    .catch((err) => {
      console.log(err)
    });
};

onMounted(async () => {
  if (props.expand) {
    collapse.value = props.title!
  }
  fetchFilters()
})
</script>

<template>
  <el-collapse v-model="collapse"
               accordion>
    <el-collapse-item :name="title">
      <template #title>
        <div class="title">
          {{ title }}
          <span> ({{ items.length }})</span>
        </div>
      </template>
      <el-input v-model="search"
                clearable
                class="search"
                :suffix-icon="Search" />
      <UButton :icon="orderBy != 1 ? 'i-fa6-solid-arrow-down-a-z' : 'i-fa6-solid-arrow-down-z-a'"
               color="gray"
               variant="ghost"
               @click="changeOrderAlphabet"
               :class="{ fade: orderBy > 1 }" />
      <UButton :icon="orderBy != 3 ? 'i-fa6-solid-arrow-down-9-1' : 'i-fa6-solid-arrow-down-1-9'"
               color="gray"
               variant="ghost"
               @click="changeOrderCount"
               :class="{ fade: orderBy < 2 }" />
      <UButton icon="i-fa6-solid-arrows-up-down"
               color="gray"
               variant="ghost"
               @click="negateSelection" />
      <UButton v-if="searchedItemsChecked.length == searchedItems.length"
               icon="i-fa6-solid-trash-can"
               color="gray"
               variant="ghost"
               @click="clearSelection" />
      <UButton v-else
               icon="i-fa6-solid-check-double"
               color="gray"
               variant="ghost"
               @click="selectAll" />
      <UButton icon="i-fa6-solid-clipboard"
               color="gray"
               variant="ghost"
               @click="copyToClipboard" />
      <el-checkbox-group v-model="props.filters![field!]"
                         class="scrollable"
                         @change="checkboxChanged()"
                         :disabled="disabled">
        <template v-for="item in searchedItems">
          <div>
            <el-checkbox :label="item.name">
              <span v-if="props.type && props.type == 'boolean'">
                <UIcon v-if="item.name === false"
                       name="i-fa6-solid-xmark"
                       style="color: red" />
                <UIcon v-if="item.name === true"
                       name="i-fa6-solid-check"
                       style="color: green" />
              </span>
              <span v-else>{{ item.name }}</span>
              ({{ item.count }})
            </el-checkbox>
          </div>
        </template>
      </el-checkbox-group>
    </el-collapse-item>
    <el-checkbox-group v-if="props.filters![field!]"
                       v-model="props.filters![field!]"
                       @change="checkboxChanged()"
                       :disabled="disabled">
      <template v-for="item in items">
        <div>
          <el-checkbox v-if="props.filters![field!].includes(item.name)"
                       :label="item.name">
            <span v-if="props.type && props.type == 'boolean'">
              <UIcon v-if="item.name === false"
                     name="i-fa6-solid-xmark" />
              <UIcon v-if="item.name === true"
                     name="i-fa6-solid-check" />
            </span>
            <span v-else>{{ item.name }}</span>
            ({{ item.count }})
          </el-checkbox>
        </div>
      </template>
    </el-checkbox-group>
  </el-collapse>
</template>

<style scoped>
.title {
  font-size: 15px;
}

.scrollable {
  overflow: auto;
  max-height: 280px;
}

.scrollable::-webkit-scrollbar {
  height: 0px;
  width: 8px;
  border: 1px solid #fff;
}

.scrollable::-webkit-scrollbar-track {
  border-radius: 0;
  background: #eeeeee;
}

.scrollable::-webkit-scrollbar-thumb {
  border-radius: 0;
  background: #b0b0b0;
}

.search {
  margin-bottom: 5px;
}

.el-input {
  width: 50%;
}

.el-button {
  margin-bottom: 5px;
  padding: 0 5px;
  margin-left: 10px
}

.el-button+.el-button {
  margin-left: 0;
}

.fade {
  opacity: 0.33;
}

button {
  padding: 0px 2px;
  vertical-align: text-bottom;
}
</style>