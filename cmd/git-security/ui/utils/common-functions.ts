export interface DataKeyItem {
  value: string;
  link: string;
  displayValue: string;
}

export const allKnownDataKeys = ref<DataKeyItem[]>([
  { value: "created_at", link: "", displayValue: "Repository Creation Time" },
  {
    value: "default_branch.name",
    link: "",
    displayValue: "Default Branch Name",
  },
  {
    value: "default_branch.branch_protection_rule.pattern",
    link: "",
    displayValue: "Branch Protection Rule: Name Pattern",
  },
  {
    value: "default_branch.branch_protection_rule.allows_deletion",
    link: "",
    displayValue: "Branch Protection Rule: Allow deletions",
  },
  {
    value: "default_branch.branch_protection_rule.allows_force_pushes",
    link: "",
    displayValue: "Branch Protection Rule: Allow force pushes",
  },
  {
    value: "default_branch.branch_protection_rule.dismisses_stale_reviews",
    link: "",
    displayValue: "Branch Protection Rule: Restrict who can dismiss pull request reviews",
  },
  {
    value: "default_branch.branch_protection_rule.is_admin_enforced",
    link: "",
    displayValue: "Branch Protection Rule: Do not allow bypassing the above settings",
  },
  {
    value: "default_branch.branch_protection_rule.require_last_push_approval",
    link: "",
    displayValue: "Branch Protection Rule: Require approval of the most recent reviewable push",
  },
  {
    value:
      "default_branch.branch_protection_rule.required_approving_review_count",
    link: "",
    displayValue: "Branch Protection Rule: Require approvals",
  },
  {
    value: "default_branch.branch_protection_rule.required_status_checks",
    link: "",
    displayValue: "Branch Protection Rule: Status checks that are required",
  },
  {
    value: "default_branch.branch_protection_rule.requires_approving_reviews",
    link: "",
    displayValue: "Branch Protection Rule: Require a pull request before merging",
  },
  {
    value: "default_branch.branch_protection_rule.requires_code_owner_reviews",
    link: "",
    displayValue: "Branch Protection Rule: Require review from Code Owners",
  },
  {
    value: "default_branch.branch_protection_rule.requires_commit_signatures",
    link: "",
    displayValue: "Branch Protection Rule: Require signed commits",
  },
  {
    value:
      "default_branch.branch_protection_rule.requires_conversation_resolution",
    link: "",
    displayValue: "Branch Protection Rule: Require conversation resolution before merging",
  },
  {
    value: "default_branch.branch_protection_rule.requires_linear_history",
    link: "",
    displayValue: "Branch Protection Rule: Require linear history",
  },
  {
    value: "default_branch.branch_protection_rule.requires_status_checks",
    link: "",
    displayValue: "Branch Protection Rule: Require status checks to pass before merging",
  },
  {
    value:
      "default_branch.branch_protection_rule.requires_strict_status_checks",
    link: "",
    displayValue: "Branch Protection Rule: Require branches to be up to date before merging",
  },
  {
    value: "default_branch.branch_protection_rule.retricts_pushes",
    link: "",
    displayValue: "Branch Protection Rule: Restrict who can dismiss pull request reviews",
  },
  {
    value: "default_branch.branch_protection_rule.retricts_review_dismissals",
    link: "",
    displayValue: "Branch Protection Rule: Do not allow bypassing the above settings",
  },
  {
    value: "delete_branch_on_merge",
    link: "",
    displayValue: "Delete Branch on Merge",
  },
  { value: "disk_usage", link: "", displayValue: "Disk Usage" },
  { value: "full_name", link: "", displayValue: "Repository Full Name" },
  { value: "is_archived", link: "", displayValue: "Is Repository Archived" },
  { value: "is_disabled", link: "", displayValue: "Is Repository Disabled" },
  { value: "is_empty", link: "", displayValue: "Is Repository Empty" },
  { value: "is_locked", link: "", displayValue: "Is Repository Locked" },
  { value: "is_private", link: "", displayValue: "Is Repository Private" },
  {
    value: "last_committed_at",
    link: "",
    displayValue: "Repository Last Commit Time",
  },
  {
    value: "merge_commit_allowed",
    link: "",
    displayValue: "Is Merge Commit Allowed",
  },
  { value: "name", link: "", displayValue: "Repository Name" },
  { value: "owner.login", link: "", displayValue: "Repository Organization" },
  {
    value: "primary_language.name",
    link: "",
    displayValue: "Repository Primary Language",
  },
  {
    value: "pull_requests.total_count",
    link: "",
    displayValue: "Pull Requests Total Count",
  },
  {
    value: "rebase_merge_allowed",
    link: "",
    displayValue: "Is Rebase Merge Allowed",
  },
  { value: "refs.total_count", link: "", displayValue: "Refs Total Count" },
  {
    value: "squash_merge_allowed",
    link: "",
    displayValue: "Is Squash Merge Allowed",
  },
  { value: "updated_at", link: "", displayValue: "Repository Update Time" },
]);

export const columnDataKeys = (queryString: string, cb: any) => {
  const results = queryString
    ? allKnownDataKeys.value.filter(createFilter(queryString))
    : allKnownDataKeys.value.map((item) => ({
        value: item.value,
        displayValue: item.displayValue ? item.displayValue : item.value, // Use displayValue if available, otherwise use value
      }));
  // call callback function to return suggestions
  cb(results);
};

export const createFilter = (queryString:string) => {
  // Convert the query string to lowercase for case-insensitive matching
  const query = queryString.toLowerCase();
  return (item: DataKeyItem) => {
    // Check if the displayValue or value of the item contains the query string
    return (
      item.displayValue.toLowerCase().includes(query) ||
      item.value.toLowerCase().includes(query)
    );
  };
};

export const dotGet = (obj: Record<string, any>, str: string): any => {
  let s = str.split(".");
  for (var i = 0; i < s.length; i++) {
    obj = obj[s[i]];
    if (obj == undefined) {
      return undefined;
    }
  }
  return obj;
}
