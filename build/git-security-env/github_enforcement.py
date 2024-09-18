#!/usr/local/bin/python

import json
import os
import requests
import sys
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


def get_boolean_env_var(name):
    v = os.environ.get(name)
    if v is None:
        return None
    v = v.lower()
    return any(v == ans for ans in ["true", "t", "yes", "y", "1"])


def get_list_env_var(name):
    v = os.environ.get(name)
    if v is None:
        return None
    return v.split(",")


def is_bpr_option_equal(bpr, key, from_env, tranformation=None):
    if from_env is None:
        return True  # we don't need to compare since we are not changing it
    if tranformation is not None:
        from_env = tranformation(from_env)
    if key in bpr and bpr[key] == from_env:
        return True
    print("%s diffs %s <> %s" % (key, bpr.get(key), from_env))


def is_repo_list_equal(repo, key, from_env):
    if from_env is None:
        return True  # we don't need to compare since we are not changing it
    if key in repo and len(repo[key]) == len(from_env):
        for e in repo[key]:
            if e not in from_env:
                print("%s diffs %s <> %s" % (key, repo.get(key), from_env))
                return False
        return True
    print("%s diffs %s <> %s" % (key, repo.get(key), from_env))


GITHUB_HOST = os.environ["GITHUB_HOST"]
GH_TOKEN = os.environ["GH_TOKEN"]
GIT_REPO_JSON = os.environ["GIT_REPO_JSON"]
repo = json.loads(GIT_REPO_JSON)

GITHUB_BRANCH_PROTECTION_RULE = get_boolean_env_var("GITHUB_BRANCH_PROTECTION_RULE")
GITHUB_BPR_REQUIRED_PR = get_boolean_env_var("GITHUB_BPR_REQUIRED_PR")
GITHUB_BPR_APPROVING_REVIEW_COUNT = os.environ.get("GITHUB_BPR_APPROVING_REVIEW_COUNT")
GITHUB_BPR_DISMISS_STALE_REVIEWS = get_boolean_env_var(
    "GITHUB_BPR_DISMISS_STALE_REVIEWS"
)
GITHUB_BPR_REQUIRED_STATUS_CHECKS = get_boolean_env_var(
    "GITHUB_BPR_REQUIRED_STATUS_CHECKS"
)
GITHUB_BPR_ENFORCE_ADMINS = get_boolean_env_var("GITHUB_BPR_ENFORCE_ADMINS")
GITHUB_BPR_CONVERSATION_RESOLUTION = get_boolean_env_var(
    "GITHUB_BPR_CONVERSATION_RESOLUTION"
)
GITHUB_BPR_ALLOW_FORCE_PUSHES = get_boolean_env_var("GITHUB_BPR_ALLOW_FORCE_PUSHES")
GITHUB_BPR_ALLOW_DELETIONS = get_boolean_env_var("GITHUB_BPR_ALLOW_DELETIONS")
GITHUB_BPR_BYPASS_PR_USERS = get_list_env_var("GITHUB_BPR_BYPASS_PR_USERS")

# check what we need to do
# if GITHUB_BRANCH_PROTECTION_RULE is true / none
#   rule doesn't exist and GITHUB_BRANCH_PROTECTION_RULE is true => create one with the options
#   rule exists => check each option with the existing values => update if needed
# if GITHUB_BRANCH_PROTECTION_RULE is false
#   rule exists => delete the rule with graphql

bpr = repo["default_branch"]["branch_protection_rule"]

m = """
mutation deleteBranchProtectionRule($ruleId: ID!) {
    deleteBranchProtectionRule(input: {
        branchProtectionRuleId: $ruleId,
    }) {
        clientMutationId
    }
}
"""
variables = {
    "repoId": repo["id"],
    "pattern": repo["default_branch"]["name"],
    "ruleId": bpr["id"],
    "requiresApprovingReviews": GITHUB_BPR_REQUIRED_PR,
    "requiredApprovingReviewCount": (
        None
        if GITHUB_BPR_APPROVING_REVIEW_COUNT is None
        else int(GITHUB_BPR_APPROVING_REVIEW_COUNT)
    ),
    "dismissesStaleReviews": GITHUB_BPR_DISMISS_STALE_REVIEWS,
    "requiresStatusChecks": GITHUB_BPR_REQUIRED_STATUS_CHECKS,
    "isAdminEnforced": GITHUB_BPR_ENFORCE_ADMINS,
    "requiresConversationResolution": GITHUB_BPR_CONVERSATION_RESOLUTION,
    "allowsForcePushes": GITHUB_BPR_ALLOW_FORCE_PUSHES,
    "allowsDeletions": GITHUB_BPR_ALLOW_DELETIONS,
    "bypassPullRequestActorIds": GITHUB_BPR_BYPASS_PR_USERS,
}

if GITHUB_BRANCH_PROTECTION_RULE is None or GITHUB_BRANCH_PROTECTION_RULE:
    if bpr["id"] == "" and GITHUB_BRANCH_PROTECTION_RULE:
        # create case
        m = """
mutation createBranchProtectionRule(
    $repoId: ID!
    $pattern: String!
    $requiresApprovingReviews: Boolean
    $requiredApprovingReviewCount: Int
    $dismissesStaleReviews: Boolean
    $requiresStatusChecks: Boolean
    $isAdminEnforced: Boolean
    $requiresConversationResolution: Boolean
    $allowsForcePushes: Boolean
    $allowsDeletions: Boolean
    $bypassPullRequestActorIds: [ID!]
) {
    createBranchProtectionRule(input: {
        repositoryId: $repoId
        pattern: $pattern
        requiresApprovingReviews: $requiresApprovingReviews
        requiredApprovingReviewCount: $requiredApprovingReviewCount
        dismissesStaleReviews: $dismissesStaleReviews
        requiresStatusChecks: $requiresStatusChecks
        isAdminEnforced: $isAdminEnforced
        requiresConversationResolution: $requiresConversationResolution
        allowsForcePushes: $allowsForcePushes
        allowsDeletions: $allowsDeletions
        bypassPullRequestActorIds: $bypassPullRequestActorIds
    }) {
        branchProtectionRule {
            id
        }
    }
}
"""
    elif bpr["id"] != "":
        # update case

        # check diff first
        if (
            is_bpr_option_equal(
                bpr, "requires_approving_reviews", GITHUB_BPR_REQUIRED_PR
            )
            and is_bpr_option_equal(
                bpr, "dismisses_stale_reviews", GITHUB_BPR_DISMISS_STALE_REVIEWS
            )
            and is_bpr_option_equal(
                bpr, "requires_status_checks", GITHUB_BPR_REQUIRED_STATUS_CHECKS
            )
            and is_bpr_option_equal(bpr, "is_admin_enforced", GITHUB_BPR_ENFORCE_ADMINS)
            and is_bpr_option_equal(
                bpr,
                "requires_conversation_resolution",
                GITHUB_BPR_CONVERSATION_RESOLUTION,
            )
            and is_bpr_option_equal(
                bpr, "allows_force_pushes", GITHUB_BPR_ALLOW_FORCE_PUSHES
            )
            and is_bpr_option_equal(bpr, "allows_deletion", GITHUB_BPR_ALLOW_DELETIONS)
            and is_bpr_option_equal(
                bpr,
                "required_approving_review_count",
                GITHUB_BPR_APPROVING_REVIEW_COUNT,
                lambda x: int(x),
            )
            and is_repo_list_equal(
                repo, "bypass_pull_request_user_ids", GITHUB_BPR_BYPASS_PR_USERS
            )
        ):
            sys.exit(0)
        m = """
mutation UpdateBranchProtectionRule(
    $ruleId: ID!
    $requiresApprovingReviews: Boolean
    $requiredApprovingReviewCount: Int
    $dismissesStaleReviews: Boolean
    $requiresStatusChecks: Boolean
    $isAdminEnforced: Boolean
    $requiresConversationResolution: Boolean
    $allowsForcePushes: Boolean
    $allowsDeletions: Boolean
    $bypassPullRequestActorIds: [ID!]
) {
    updateBranchProtectionRule(input: {
        branchProtectionRuleId: $ruleId
        requiresApprovingReviews: $requiresApprovingReviews
        requiredApprovingReviewCount: $requiredApprovingReviewCount
        dismissesStaleReviews: $dismissesStaleReviews
        requiresStatusChecks: $requiresStatusChecks
        isAdminEnforced: $isAdminEnforced
        requiresConversationResolution: $requiresConversationResolution
        allowsForcePushes: $allowsForcePushes
        allowsDeletions: $allowsDeletions
        bypassPullRequestActorIds: $bypassPullRequestActorIds
    }) {
        branchProtectionRule {
            id
        }
    }
}
"""
    else:
        sys.exit(0)

url = "https://%s/api/graphql" % (GITHUB_HOST,)
headers = {"Authorization": f"Bearer {GH_TOKEN}", "Content-Type": "application/json"}
response = requests.post(
    url,
    headers=headers,
    json={"query": m, "variables": variables},
    verify=False,
)
print(response.text)
