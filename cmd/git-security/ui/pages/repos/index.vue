<script setup lang="ts">
import {
  ElCheckbox,
  ElInput,
  ElLink,
  ElTooltip,
  TableV2FixedDir,
  TableV2SortOrder,
} from "element-plus";
import type {
  CheckboxValueType,
  Column,
  RowClassNameGetter,
  SortBy,
  TransferDataItem,
} from "element-plus";
import { Loading as LoadingIcon } from "@element-plus/icons-vue";
import Icon from "#ui/components/elements/Icon.vue";
import UButton from "#ui/components/elements/Button.vue";
import {
  showConfirmationDialog,
  showNotification,
  actionsConfirmationDialog,
} from "@/common-functions";

type ColumnType =
  | "string"
  | "number"
  | "boolean"
  | "array"
  | "date"
  | "reposcore";
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
};

const loading = ref(false);
const filters = ref<Record<string, string[]>>({});
const negates = ref<Record<string, boolean>>({});
const includeZeroTimes = ref<Record<string, boolean>>({});
const types = ref<Record<string, string>>({});
const filtersOrder = ref([]);
const updateFilters = (field: string) => {
  fetchRepos();
};
var lastCheckboxCheckedIndex = -1;

const transform = (f: Record<string, string[]>) => {
  let results = [];
  for (let filter in f) {
    if (f[filter].length > 0) {
      results.push({
        type: types.value[filter],
        field: filter,
        values: f[filter],
        negate: negates.value[filter] || false,
        include_zero_time:
          filter in includeZeroTimes.value
            ? includeZeroTimes.value[filter]
            : true,
      });
    }
  }
  return results;
};

const fetchRepos = () => {
  loading.value = true;
  $fetch("/api/v1/repos?archived=" + (uiData.showArchived ? "true" : "false"), {
    method: "POST",
    body: {
      filters: transform(filters.value),
    },
    onResponse({ response }) {
      loading.value = false;
      repos_table.data = response._data;
      repos_table.data = useOrderBy(
        repos_table.data,
        ["full_name"],
        [TableV2SortOrder.ASC]
      );
      repos_table.originalData = repos_table.data;
      repos_table.dataMap.clear();
      for (let i = 0; i < repos_table.originalData.length; i++) {
        repos_table.dataMap.set(repos_table.originalData[i].id, i);
      }
      repos_table_search.text = "";
      repos_table.selected.clear();
      lastCheckboxCheckedIndex = -1;
    },
  });
};

const refreshFilter = async () => {
  Object.keys(filters.value).forEach((key) => delete filters.value[key]);
  filtersOrder.value.splice(0);
  fetchRepos();
};

const exportToCSV = async () => {
  $fetch(
    "/api/v1/repos?csv=true&archived=" +
      (uiData.showArchived ? "true" : "false"),
    {
      method: "POST",
      body: {
        filters: transform(filters.value),
      },
      onResponse({ response }) {
        const url =
          "data:text/csv;charset=utf-8," + encodeURIComponent(response._data);
        const a = document.createElement("a");
        a.href = url;
        a.setAttribute("download", "repos.csv");
        a.click();
      },
    }
  );
};

const getDefaultColumns = (): Column<any>[] => [
  {
    key: "selection",
    width: 25,
    cellRenderer: ({ rowData, rowIndex }) => {
      const onChange = (value: CheckboxValueType) => {
        if (value) {
          repos_table.selected.set(rowData["id"], true);
        } else {
          repos_table.selected.delete(rowData["id"]);
        }
      };

      const shiftClick = (e: PointerEvent) => {
        // setTimeout here to let the checkbox onChange to run first
        setTimeout(() => {
          if (repos_table.selected.get(rowData["id"])) {
            if (e.shiftKey) {
              if (lastCheckboxCheckedIndex > -1) {
                let range = [rowIndex, lastCheckboxCheckedIndex];
                if (rowIndex > lastCheckboxCheckedIndex) {
                  range = [lastCheckboxCheckedIndex, rowIndex];
                }
                for (var i in repos_table.data) {
                  if (parseInt(i) >= range[0] && parseInt(i) <= range[1]) {
                    const rowData = repos_table.data[i];
                    repos_table.selected.set(rowData["id"], true);
                  }
                }
              }
            }
            lastCheckboxCheckedIndex = rowIndex;
          } else {
            lastCheckboxCheckedIndex = -1;
          }
        }, 0);
      };

      return h(
        "div",
        {
          style: { padding: "5px 0" },
        },
        h(ElCheckbox, {
          onChange: onChange,
          modelValue: repos_table.selected.has(rowData["id"]),
          onClick: shiftClick,
        })
      );
    },
    headerCellRenderer: () => {
      const onChange = (value: CheckboxValueType) => {
        for (var i in repos_table.data) {
          const rowData = repos_table.data[i];
          if (value) {
            repos_table.selected.set(rowData["id"], true);
          } else {
            repos_table.selected.delete(rowData["id"]);
          }
        }
      };
      return h(ElCheckbox, {
        modelValue:
          repos_table.selected.size > 0 &&
          repos_table.selected.size == repos_table.data.length,
        indeterminate:
          repos_table.selected.size > 0 &&
          repos_table.selected.size != repos_table.data.length,
        onChange: onChange,
      });
    },
    fixed: TableV2FixedDir.LEFT,
  },
  {
    title: "",
    key: "row",
    width: 80,
    align: "center",
    fixed: TableV2FixedDir.LEFT,
    cellRenderer: ({ rowIndex }: any) => h("span", `${rowIndex + 1}`),
  },
  {
    title: "Repo Name",
    key: "full_name",
    dataKey: "full_name",
    width: 400,
    sortable: true,
    fixed: TableV2FixedDir.LEFT,
    cellRenderer: ({ cellData, rowData }) =>
      h(
        "div",
        {
          style: { padding: "5px 0" },
        },
        h(
          ElLink,
          {
            href: `https://${rowData["github_host"]}/${cellData}`,
            target: "_blank",
            underline: false,
            onClick: (e) => e.stopPropagation(),
          },
          () => cellData
        )
      ),
    headerCellRenderer: () => {
      return h("div", {}, [
        h("span", {}, "Repo Name"),
        h(
          "span",
          {
            style: {
              width: "250px",
              display: "inline-block",
              "margin-left": "10px",
              "margin-right": "10px",
            },
          },
          h(ElInput, {
            onClick: (e: any) => {
              e.stopPropagation();
            },
            onInput: (v) => {
              repos_table_search.text = v;
              refresh_repos_table_data();
              lastCheckboxCheckedIndex = -1;
            },
            modelValue: repos_table_search.text,
          })
        ),
        h(
          "span",
          {
            style: {
              "vertical-align": "bottom",
              "margin-left": "-40px",
              "margin-right": "10px",
            },
          },
          h(UButton, {
            onClick: (e: any) => {
              e.stopPropagation();
              repos_table_search.text = "";
              refresh_repos_table_data();
            },
            color: "gray",
            circle: true,
            variant: "ghost",
            size: "sm",
            icon: "i-heroicons-x-mark-20-solid",
          })
        ),
      ]);
    },
  },
];

const refresh_repos_table_data = () => {
  if (repos_table_search.text != "") {
    repos_table.data = useFilter(repos_table.originalData, (row: any) => {
      return (
        row.full_name
          .toLowerCase()
          .indexOf(repos_table_search.text.toLowerCase()) > -1
      );
    });
  } else {
    repos_table.data = repos_table.originalData;
  }
};

const uiData = reactive({
  filterCCs: <ColumnConfig[]>[],
  availableFilters: <ColumnConfigTransfer[]>[],
  availableColumns: <ColumnConfigTransfer[]>[],
  allCCsMap: <Record<string, ColumnConfig>>{},
  checkedFilters: <string[]>[],
  selectedFilters: <string[]>[],
  checkedColumns: <string[]>[],
  selectedColumns: <string[]>[],
  selectedFiltersExpanded: <Record<string, boolean>>{},
  showArchived: false,
});

const fetchAllColumns = () => {
  $fetch("/api/v1/columns", {
    method: "GET",
    onResponse({ response }) {
      uiData.availableFilters = [];
      uiData.availableColumns = [];
      uiData.allCCsMap = {};
      response._data.forEach((cc: ColumnConfig) => {
        uiData.availableFilters.push({
          key: cc.id,
          label: cc.title,
        });
        uiData.availableColumns.push({
          key: cc.id,
          label: cc.title,
        });

        uiData.allCCsMap[cc.id] = cc;
      });

      uiData.availableFilters.sort((a, b) => a.label.localeCompare(b.label));
      uiData.availableColumns.sort((a, b) => a.label.localeCompare(b.label));

      fetchUserView();
    },
  });
};

const repos_table_search = reactive({
  text: "",
});

const repos_table = reactive({
  selectedDeviceID: null,
  rowKey: "id",
  columns: getDefaultColumns(),
  dataMap: new Map<string, number>(),
  originalData: <any>[],
  data: <any>[],
  selected: new Map(),
  sortState: ref<SortBy>({
    key: "full_name",
    order: TableV2SortOrder.ASC,
  }),
  onSort: (sortBy: SortBy) => {
    repos_table.data = useOrderBy(
      repos_table.data,
      [sortBy.key],
      [sortBy.order]
    );
    repos_table.originalData = useOrderBy(
      repos_table.originalData,
      [sortBy.key],
      [sortBy.order]
    );
    repos_table.sortState = sortBy;
  },
  rowClass: ({ rowIndex }: Parameters<RowClassNameGetter<any>>[0]) => {
    if (rowIndex % 2 === 1) {
      return "zebra";
    }
    return "";
  },
});

const actionAPI = async (
  api: string,
  actionLabel: string,
  confirmLabel?: string,
  cancelLabel?: string
) => {
  try {
    let confirmed;
    if (actionLabel == "Add Default Branch Protection Rule") {
      confirmed = await showConfirmationDialog(
        `Are you sure you want to perform the action:\n ${actionLabel} ?`
      );
    } else {
      confirmed = await actionsConfirmationDialog(
        `${actionLabel}`,
        confirmLabel,
        cancelLabel
      );
    }
    if (confirmed != undefined) {
      $fetch(api, {
        method: "POST",
        body: {
          ids: Array.from(repos_table.selected.keys()),
          updateValue: confirmed,
        },
        onResponse: ({ response }) => {
          if (response.status == 200) {
            showNotification("success");
          } else {
            showNotification("error");
          }
        },
      });
    }
  } catch (error) {
    console.error("Error in confirmation dialog:", error);
  }
};

const actions = [
  [
    {
      label: "Add Default Branch Protection Rule",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/add-branch-protection-rule",
          "Add Default Branch Protection Rule"
        ),
    },
  ],
  [
    {
      label: "Requires PR",
      click: () => actionAPI("/api/v1/repos/action/requires-pr", "Requires PR"),
    },
  ],
  [
    {
      label: "How many Approvers?",
      click: () => {
        const options = Array.from({ length: 7 }, (_, i) => ({
          value: i.toString(),
          label: i.toString(),
        }));

        const selectElement = document.createElement("select");
        selectElement.id = "approversSelect"; // Add an id to the select element
        options.forEach((option) => {
          const optionElement = document.createElement("option");
          optionElement.value = option.value;
          optionElement.text = option.label;
          selectElement.appendChild(optionElement);
        });

        const message = `
          <div>
            Choose Number of Approvers : ${selectElement.outerHTML}
          </div>
        `;
        ElMessageBox.alert(message, "Number of Approvers", {
          dangerouslyUseHTMLString: true,
          confirmButtonText: "Submit",
          cancelButtonText: "Cancel",
          callback: (action, instance) => {
            if (action == "confirm") {
              const selectElementInMessageBox =
                document.getElementById("approversSelect"); // Get the select element from the MessageBox
              let value = selectElementInMessageBox
                ? (selectElementInMessageBox as HTMLSelectElement).value
                : "";
              if (value) {
                let count = Number(value);
                $fetch("/api/v1/repos/action/required-approving-review-count", {
                  method: "POST",
                  body: {
                    ids: Array.from(repos_table.selected.keys()),
                    updateValue: count,
                  },
                  onResponse: ({ response }) => {
                    if (response.status == 200) {
                      showNotification("success");
                    } else {
                      showNotification("error");
                    }
                  },
                });
              }
            }
          },
        });
      },
    },
  ],
  [
    {
      label: "Requires Approving Review Count",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/required-approving-review-count",
          "Requires Approving Review Count"
        ),
    },
  ],
  [
    {
      label: "Dismiss Stale Reviews",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/dismisses-stale-reviews",
          "Dismiss Stale Review"
        ),
    },
  ],
  [
    {
      label: "Requires Code Owner Reviews",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/requires-code-owner-reviews",
          "Requires Code Owner Reviews"
        ),
    },
  ],
  [
    {
      label: "Requires Status Checks",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/requires-status-checks",
          "Requires Status Checks"
        ),
    },
  ],
  [
    {
      label: "Requires Strict Status Checks",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/requires-strict-status-checks",
          "Requires Strict Status Checks"
        ),
    },
  ],
  [
    {
      label: "Requires Conversation Resolution",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/requires-conversation-resolution",
          "Requires Conversation Resolution"
        ),
    },
  ],
  [
    {
      label: "Requires Commit Signatures",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/requires-commit-signatures",
          "Requires Commit Signatures"
        ),
    },
  ],
  [
    {
      label: "Admin Enforced",
      click: () =>
        actionAPI("/api/v1/repos/action/admin-enforced", "Admin Enforced"),
    },
  ],
  [
    {
      label: "Allow Force Pushes",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/allows-force-pushes",
          "Allow Force Pushes"
        ),
    },
  ],
  [
    {
      label: "Allow Deletions",
      click: () =>
        actionAPI("/api/v1/repos/action/allows-deletions", "Allow Deletions"),
    },
  ],
  [
    {
      label: "Archive/Unarchive Repos",
      click: () =>
        actionAPI(
          "/api/v1/repos/action/archive-repo",
          "Archive/Unarchive Repos",
          "Archive",
          "Unarchive"
        ),
    },
  ],
  [
    {
      label: "Pre-receive Hook",
      click: () => {
        hookName.value = "";
        hookDialogVisible.value = true;
      },
    },
  ],
  [
    {
      label: "Update Owner",
      click: () => {
        selectedOwnerID.value = "";
        ownerDialogVisible.value = true;
      },
    },
    {
      label: "Delete Owner",
      click: async () => {
        try {
          const confirmed = await showConfirmationDialog(
            `Are you sure you want to delete the owner for selected repos ?`
          );
          if (confirmed) {
            var ids = Array.from(repos_table.selected.keys());
            $fetch(`/api/v1/repos/action/delete-owner`, {
              method: "POST",
              body: ids,
              onResponse: ({ response }) => {
                if (response.status == 200) {
                  showNotification("success");
                } else {
                  showNotification("error");
                }
              },
            });
          }
        } catch (error) {
          console.error("An error occurred:", error);
        }
      },
    },
  ],
];

const handleWebSocketMessage = (event: MessageEvent) => {
  const updatedRepo = JSON.parse(event.data);
  let i = repos_table.dataMap.get(updatedRepo.id);
  if (i != undefined) {
    repos_table.originalData[i] = updatedRepo;
    refresh_repos_table_data();
  }
};

const setupWebSocket = () => {
  const ws = new WebSocket(location.origin.replace(/^http/, "ws") + "/ws");

  ws.onopen = () => {
    console.log("WebSocket connection opened");
  };

  ws.onmessage = (event) => handleWebSocketMessage(event);

  ws.onclose = (event) => {
    console.error("WebSocket connection closed:", event.code, event.reason);
  };

  ws.onerror = (error) => {
    console.error("WebSocket error:", error);
  };
};

const dialog = ref(false);
const handleClose = () => {
  dialog.value = false;
};
const onSubmit = () => {
  var filters = [];
  for (const f of uiData.selectedFilters) {
    filters.push({
      id: f,
      filter_expanded:
        f in uiData.selectedFiltersExpanded
          ? uiData.selectedFiltersExpanded[f]
          : false,
    });
  }
  $fetch("/api/v1/userview", {
    method: "PUT",
    body: {
      show_archived: uiData.showArchived,
      filters: filters,
      columns: uiData.selectedColumns,
    },
    onResponse({ response }) {
      fetchAllColumns();
      dialog.value = false;
    },
  });
};

const renderFunc = (h: any, option: TransferDataItem) => {
  if (uiData.selectedFilters.indexOf(option.key) > -1) {
    return h("span", null, [
      option.label,
      h(UButton, {
        onClick: (e: any) => {
          e.stopPropagation();
          if (uiData.selectedFiltersExpanded[option.key] == undefined) {
            uiData.selectedFiltersExpanded[option.key] = false;
          }
          uiData.selectedFiltersExpanded[option.key] =
            !uiData.selectedFiltersExpanded[option.key];
        },
        color: "gray",
        circle: true,
        variant: "ghost",
        size: "sm",
        icon: uiData.selectedFiltersExpanded[option.key]
          ? "i-heroicons-arrows-pointing-out-20-solid"
          : "i-heroicons-arrows-pointing-in-20-solid",
        style:
          "margin-left: 10px; padding: 0px; vertical-align: middle; margin-top: -2px",
      }),
    ]);
  } else {
    return h("span", null, option.label);
  }
};

type ColumnConfigTransfer = {
  key: string;
  label: string;
};

type Filter = {
  id: string;
  filter_expanded: boolean;
};

type UserView = {
  show_archived: boolean;
  filters: Filter[];
  columns: string[];
};

const fetchUserView = () => {
  $fetch("/api/v1/userview", {
    method: "GET",
    onResponse({ response }) {
      uiData.selectedFilters = [];
      uiData.selectedFiltersExpanded = {};
      uiData.selectedColumns = [];
      uiData.filterCCs = [];
      uiData.showArchived = false;
      types.value = {};

      // give it a chance for the filters to refresh
      setTimeout(() => {
        var uv = <UserView>response._data;
        uiData.showArchived = uv.show_archived;

        for (const f of uv.filters) {
          if (f.id in uiData.allCCsMap) {
            uiData.selectedFilters.push(f.id);
            uiData.selectedFiltersExpanded[f.id] = f.filter_expanded;

            let cc = uiData.allCCsMap[f.id];
            cc.filter_expanded = f.filter_expanded;
            uiData.filterCCs.push(cc);
            types.value[cc.key] = cc.type;
          }
        }

        repos_table.columns = getDefaultColumns();
        for (const id of uv.columns) {
          if (id in uiData.allCCsMap) {
            let cc = uiData.allCCsMap[id];
            uiData.selectedColumns.push(id);

            let c: Column<any> = {
              title: cc.title,
              key: cc.key,
              dataKey: cc.key,
              width: cc.width,
              sortable: true,
            };
            if (cc.type != "string" && cc.type != "array") {
              c["align"] = "center";
            }

            const getContent = (rowData: any, cellData: any) => {
              if (cc.type == "boolean") {
                return h(Icon, {
                  name: cellData ? "i-fa6-solid-check" : "i-fa6-solid-xmark",
                  style: cellData ? "color: green" : "color: red",
                });
              } else if (cc.type == "array") {
                return cellData && Array.isArray(cellData)
                  ? h(
                      ElTooltip,
                      {
                        content: cellData.join(", "),
                        placement: "right",
                        "show-after": 500,
                      },
                      () =>
                        h(
                          "div",
                          {
                            style: {
                              "white-space": "nowrap",
                              overflow: "hidden",
                              "text-overflow": "ellipsis",
                              width: cc.width - 25 + "px",
                            },
                          },
                          cellData.join(", ")
                        )
                    )
                  : "";
              } else if (cc.type == "date") {
                return cellData != "0001-01-01T00:00:00Z"
                  ? h(
                      "span",
                      {
                        title: useDayjs()(cellData).local().format(),
                      },
                      useDayjs()(cellData).fromNow()
                    )
                  : h("span", "");
              } else if (cc.type == "reposcore") {
                return h(
                  "div",
                  {
                    style: {
                      "padding-top": "5px",
                      "background-color": rowData["score_color"],
                      "border-radius": "50%",
                      width: "31px",
                      height: "31px",
                    },
                  },
                  h("div", {}, cellData)
                );
              } else {
                return h("span", cellData);
              }
            };
            c["cellRenderer"] = ({ rowData, cellData }) =>
              h(
                "div",
                {
                  style: { padding: "5px 0" },
                },
                getContent(rowData, cellData)
              );

            if (cc.description) {
              const d = cc.description;
              const t = cc.title;
              c["headerCellRenderer"] = () => {
                return h(
                  "div",
                  {
                    class: "el-table-v2__header-cell-text",
                    title: d,
                  },
                  h("span", t)
                );
              };
            }
            repos_table.columns.push(c);
          }
        }

        fetchRepos();
      }, 0);
    },
  });
};

const filtersCheckedChange = (checked: any) => {
  uiData.checkedFilters = checked;
};

const filtersChange = (v: any, direction: string) => {
  if (direction == "left") {
    uiData.checkedFilters = [];
  }
};

const moveUpSelectedFilter = () => {
  let element = uiData.checkedFilters[0];
  let idx = uiData.selectedFilters.findIndex((e) => e === element);
  if (idx > 0) {
    let temp = uiData.selectedFilters[idx - 1];
    uiData.selectedFilters[idx - 1] = uiData.selectedFilters[idx];
    uiData.selectedFilters[idx] = temp;
  }
};

const moveDownSelectedFilter = () => {
  let element = uiData.checkedFilters[0];
  let idx = uiData.selectedFilters.findIndex((e) => e === element);
  if (idx >= 0 && idx < uiData.selectedFilters.length - 1) {
    let temp = uiData.selectedFilters[idx + 1];
    uiData.selectedFilters[idx + 1] = uiData.selectedFilters[idx];
    uiData.selectedFilters[idx] = temp;
  }
};

const columnsCheckedChange = (checked: any) => {
  uiData.checkedColumns = checked;
};

const columnsChange = (v: any, direction: string) => {
  if (direction == "left") {
    uiData.checkedColumns = [];
  }
};

const moveUpSelectedColumn = () => {
  let element = uiData.checkedColumns[0];
  let idx = uiData.selectedColumns.findIndex((e) => e === element);
  if (idx > 0) {
    let temp = uiData.selectedColumns[idx - 1];
    uiData.selectedColumns[idx - 1] = uiData.selectedColumns[idx];
    uiData.selectedColumns[idx] = temp;
  }
};

const moveDownSelectedColumn = () => {
  let element = uiData.checkedColumns[0];
  let idx = uiData.selectedColumns.findIndex((e) => e === element);
  if (idx >= 0 && idx < uiData.selectedColumns.length - 1) {
    let temp = uiData.selectedColumns[idx + 1];
    uiData.selectedColumns[idx + 1] = uiData.selectedColumns[idx];
    uiData.selectedColumns[idx] = temp;
  }
};

type Owner = {
  id: string;
  name: string;
  contact: string;
  notes: string;
};
const ownerDialogVisible = ref<boolean>(false);
const selectedOwnerID = ref<string>("");
const owners = ref<Owner[]>([]);
const fetchOwners = () => {
  $fetch("/api/v1/owners", {
    method: "GET",
    onResponse({ response }) {
      owners.value = response._data;
    },
  });
};

const updateOwner = () => {
  ownerDialogVisible.value = false;
  if (selectedOwnerID.value != "") {
    $fetch("/api/v1/repos/action/repo-owner", {
      method: "POST",
      body: {
        ids: Array.from(repos_table.selected.keys()),
        ownerID: selectedOwnerID.value,
      },
      onResponse: ({ response }) => {
        if (response.status == 200) {
          showNotification("success");
        } else {
          showNotification("error");
        }
      },
    });
  }
};

const hookDialogVisible = ref<boolean>(false);
const hookName = ref<string>("");
const updateHook = (enabled: boolean) => {
  hookDialogVisible.value = false;
  let v = hookName.value.trim();
  if (v != "") {
    $fetch("/api/v1/repos/action/pre-receive-hook", {
      method: "POST",
      body: {
        ids: Array.from(repos_table.selected.keys()),
        hookName: v,
        updateValue: enabled,
      },
      onResponse: ({ response }) => {
        if (response.status == 200) {
          showNotification("success");
        } else {
          showNotification("error");
        }
      },
    });
  }
};

onMounted(() => {
  setupWebSocket();
  fetchAllColumns();
  fetchOwners();
});
</script>

<template>
  <div class="common-layout">
    <el-container>
      <el-aside width="330px">
        <div class="filter-buttons">
          <el-button
            @click="dialog = true"
            circle
            size="large"
            class="filter-button"
          >
            <UIcon name="i-fa6-solid-gear" />
          </el-button>
          <el-button
            @click="exportToCSV"
            circle
            size="large"
            class="filter-button"
          >
            <UIcon name="i-fa6-solid-file-csv" />
          </el-button>
          <el-badge
            :value="repos_table.data.length"
            :max="1000000"
            class="filter-button"
            type="primary"
          >
            <el-button @click="refreshFilter" size="large" circle>
              <UIcon name="i-fa6-solid-arrows-rotate" />
            </el-button>
          </el-badge>
          <UDropdown
            :items="actions"
            mode="click"
            :popper="{ placement: 'bottom-start' }"
            v-if="repos_table.selected.size > 0"
            class="actions-button"
          >
            <UButton
              color="white"
              label="Actions"
              trailing-icon="i-heroicons-chevron-down-20-solid"
            />
            <template #item="{ item }">
              <span class="actions">{{ item.label }}</span>
            </template>
          </UDropdown>
        </div>
        <div class="filters">
          <template
            v-if="uiData.filterCCs.length > 0"
            v-for="c in uiData.filterCCs"
          >
            <Filter
              :type="c.type"
              :title="c.title"
              :field="c.key"
              :expand="c.filter_expanded"
              :filters="filters"
              :negates="negates"
              :includeZeroTimes="includeZeroTimes"
              :filtersOrder="filtersOrder"
              @updateFilters="updateFilters"
              :disabled="loading"
              :showArchived="uiData.showArchived"
            />
          </template>
        </div>
      </el-aside>
      <el-main>
        <div :style="{ height: 'calc(100vh - 150px)' }">
          <el-auto-resizer>
            <template #default="{ height, width }">
              <el-table-v2
                :row-key="repos_table.rowKey"
                :columns="repos_table.columns"
                :data="repos_table.data"
                :width="width"
                :height="height"
                fixed
                :sort-by="repos_table.sortState"
                @column-sort="repos_table.onSort"
                :row-class="repos_table.rowClass"
              >
                <template #overlay v-if="loading">
                  <div
                    class="el-loading-mask"
                    style="
                      display: flex;
                      align-items: center;
                      justify-content: center;
                    "
                  >
                    <el-icon
                      class="is-loading"
                      color="var(--el-color-primary)"
                      :size="26"
                    >
                      <loading-icon />
                    </el-icon>
                  </div>
                </template>
              </el-table-v2>
            </template>
          </el-auto-resizer>
        </div>
      </el-main>
    </el-container>

    <el-drawer
      v-model="dialog"
      title="Filter and Table Columns configuration"
      :before-close="handleClose"
      direction="ltr"
      size="70%"
    >
      <div>
        <el-transfer
          v-model="uiData.selectedFilters"
          style="
            display: flex;
            justify-content: center;
            align-items: center;
            margin: 20px;
          "
          filterable
          :render-content="renderFunc"
          :titles="['Available Filters', 'Shown Filters']"
          :format="{
            noChecked: '${total}',
            hasChecked: '${checked}/${total}',
          }"
          @right-check-change="filtersCheckedChange"
          @change="filtersChange"
          :data="uiData.availableFilters"
          target-order="push"
        >
          <template #right-footer>
            <el-button
              v-if="uiData.checkedFilters.length == 1"
              class="transfer-footer"
              size="small"
              @click="moveUpSelectedFilter"
            >
              <UIcon name="i-fa6-solid-arrow-up" />
            </el-button>
            <el-button
              v-if="uiData.checkedFilters.length == 1"
              class="transfer-footer"
              size="small"
              @click="moveDownSelectedFilter"
            >
              <UIcon name="i-fa6-solid-arrow-down" />
            </el-button>
          </template>
        </el-transfer>

        <el-transfer
          v-model="uiData.selectedColumns"
          style="
            display: flex;
            justify-content: center;
            align-items: center;
            margin: 20px;
          "
          filterable
          :titles="['Available Columns', 'Shown Columns']"
          :format="{
            noChecked: '${total}',
            hasChecked: '${checked}/${total}',
          }"
          @right-check-change="columnsCheckedChange"
          @change="columnsChange"
          :data="uiData.availableColumns"
          target-order="push"
        >
          <template #right-footer>
            <el-button
              v-if="uiData.checkedColumns.length == 1"
              class="transfer-footer"
              size="small"
              @click="moveUpSelectedColumn"
            >
              <UIcon name="i-fa6-solid-arrow-up" />
            </el-button>
            <el-button
              v-if="uiData.checkedColumns.length == 1"
              class="transfer-footer"
              size="small"
              @click="moveDownSelectedColumn"
            >
              <UIcon name="i-fa6-solid-arrow-down" />
            </el-button>
          </template>
        </el-transfer>

        <div class="actions-button">
          <el-button @click="dialog = false">Cancel</el-button>
          <el-button type="primary" @click="onSubmit">Submit</el-button>
        </div>
      </div>

      <div style="text-align: center">
        <el-switch v-model="uiData.showArchived" />
        <span style="margin-left: 10px">Show archived repositories</span>
      </div>
    </el-drawer>

    <el-dialog v-model="ownerDialogVisible" title="Update Owner" width="500">
      <el-select v-model="selectedOwnerID" size="large">
        <template #prefix>Owner</template>
        <el-option
          v-for="o in owners"
          :key="o.id"
          :label="o.name"
          :value="o.id"
        />
      </el-select>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="ownerDialogVisible = false">Cancel</el-button>
          <el-button type="primary" @click="updateOwner">Update</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="hookDialogVisible"
      title="Update Pre-receive hook"
      width="500"
    >
      <el-input v-model="hookName" clearable>
        <template #prepend>Pre-receive Hook Name</template>
      </el-input>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="updateHook(false)">Disable</el-button>
          <el-button type="primary" @click="updateHook(true)">Enable</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.example-showcase .el-table-v2__overlay {
  z-index: 9;
}

.el-aside {
  direction: rtl;
  padding: 20px 20px 20px 0;
}

.filters {
  overflow: auto;
  height: calc(100vh - 200px);
  padding-left: 20px;
}

.el-collapse {
  direction: ltr;
}

.filter-button {
  margin-bottom: 9px;
  margin-right: 8px;
  margin-left: 0px;
}

.filter-buttons {
  direction: ltr;
  padding-left: 20px;
}

.actions-button {
  float: right;
  margin-top: 4px;
}

.actions {
  text-align: left;
}
</style>
