#!/usr/local/bin/python

import json
import os
import requests

GITHUB_HOST = os.environ["GITHUB_HOST"]
GH_TOKEN = os.environ["GH_TOKEN"]
GIT_REPO_JSON = os.environ["GIT_REPO_JSON"]
repo = json.loads(GIT_REPO_JSON)

GITHUB_BRANCH_PROTECTION_RULE = (
    os.environ.get("GITHUB_BRANCH_PROTECTION_RULE", "true").lower() == "true"
)
GITHUB_BPR_REQUIRED_PR = (
    os.environ.get("GITHUB_BPR_REQUIRED_PR", "false").lower() == "true"
)
GITHUB_BPR_APPROVING_REVIEW_COUNT = os.environ.get(
    "GITHUB_BPR_APPROVING_REVIEW_COUNT", 0
)
GITHUB_BPR_DISMISS_STALE_REVIEWS = (
    os.environ.get("GITHUB_BPR_DISMISS_STALE_REVIEWS", "false").lower() == "true"
)
GITHUB_BPR_REQUIRED_STATUS_CHECKS = (
    os.environ.get("GITHUB_BPR_REQUIRED_STATUS_CHECKS", "false").lower() == "true"
)
GITHUB_BPR_ENFORCE_ADMINS = (
    os.environ.get("GITHUB_BPR_ENFORCE_ADMINS", "false").lower() == "true"
)
GITHUB_BPR_CONVERSATION_RESOLUTION = (
    os.environ.get("GITHUB_BPR_CONVERSATION_RESOLUTION", "false").lower() == "true"
)
GITHUB_BPR_ALLOW_FORCE_PUSHES = (
    os.environ.get("GITHUB_BPR_ALLOW_FORCE_PUSHES", "false").lower() == "true"
)
GITHUB_BPR_ALLOW_DELETIONS = (
    os.environ.get("GITHUB_BPR_ALLOW_DELETIONS", "false").lower() == "true"
)
GITHUB_BPR_REQUIRED_SIGNED_COMMITS = (
    os.environ.get("GITHUB_BPR_REQUIRED_SIGNED_COMMITS", "false").lower() == "true"
)

url = "https://%s/api/v3/repos/%s/branches/%s/protection" % (
    GITHUB_HOST,
    repo["full_name"],
    repo["default_branch"]["name"],
)
headers = {
    "Authorization": f"Bearer {GH_TOKEN}",
    "Accept": "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
}

if GITHUB_BRANCH_PROTECTION_RULE:
    data = {
        "restrictions": None,
    }

    if GITHUB_BPR_REQUIRED_PR:
        data["required_pull_request_reviews"] = {
            "required_approving_review_count": int(GITHUB_BPR_APPROVING_REVIEW_COUNT),
            "dismiss_stale_reviews": GITHUB_BPR_DISMISS_STALE_REVIEWS,
        }
    else:
        data["required_pull_request_reviews"] = None
    if GITHUB_BPR_REQUIRED_STATUS_CHECKS:
        data["required_status_checks"] = {"strict": True, "contexts": []}
    else:
        data["required_status_checks"] = None
    data["enforce_admins"] = GITHUB_BPR_ENFORCE_ADMINS
    data["required_conversation_resolution"] = GITHUB_BPR_CONVERSATION_RESOLUTION
    data["allow_force_pushes"] = GITHUB_BPR_ALLOW_FORCE_PUSHES
    data["allow_deletions"] = GITHUB_BPR_ALLOW_DELETIONS

    proceed = False
    bpr = repo["default_branch"]["branch_protection_rule"]
    if bpr["id"] == "":
        proceed = True
    else:
        # already existed
        bpr_required_status_checks = bpr["required_status_checks"] is not None
        if bpr["allows_force_pushes"] != GITHUB_BPR_ALLOW_FORCE_PUSHES:
            print("allows_force_pushes diffs")
            proceed = True
        elif bpr["allows_deletion"] != GITHUB_BPR_ALLOW_DELETIONS:
            print("allows_deletion diffs")
            proceed = True
        elif (
            bpr["requires_conversation_resolution"]
            != GITHUB_BPR_CONVERSATION_RESOLUTION
        ):
            print("requires_conversation_resolution diffs")
            proceed = True
        elif bpr["is_admin_enforced"] != GITHUB_BPR_ENFORCE_ADMINS:
            print("is_admin_enforced diffs")
            proceed = True
        elif bpr_required_status_checks != GITHUB_BPR_REQUIRED_STATUS_CHECKS:
            print("bpr_required_status_checks diffs")
            proceed = True
        elif (
            bpr["required_approving_review_count"] != GITHUB_BPR_APPROVING_REVIEW_COUNT
        ):
            print("required_approving_review_count diffs")
            proceed = True
        elif bpr["dismisses_stale_reviews"] != GITHUB_BPR_DISMISS_STALE_REVIEWS:
            print("dismisses_stale_reviews diffs")
            proceed = True

    # Make the API request
    if proceed:
        print("creating/updating branch protection rule")
        response = requests.put(
            url, headers=headers, data=json.dumps(data), verify=False
        )

        # Check response
        if response.status_code == 200:
            print("Branch protection rule added successfully!")
        else:
            print(f"Failed to add branch protection rule: {response.status_code}")
            print(response.json())
elif (
    not GITHUB_BRANCH_PROTECTION_RULE
    and repo["default_branch"]["branch_protection_rule"]["id"] != ""
):
    response = requests.delete(url, headers=headers, verify=False)
    if response.status_code == 204:
        print("Branch protection rule deleted successfully!")
    else:
        print(f"Failed to delete branch protection rule: {response.status_code}")
        print(response.json())

# required_signatures
if bpr["requires_commit_signatures"] != GITHUB_BPR_REQUIRED_SIGNED_COMMITS:
    url = f"{url}/required_signatures"
    if GITHUB_BPR_REQUIRED_SIGNED_COMMITS:
        response = requests.post(url, headers=headers, verify=False)
        if response.status_code == 201:
            print("Signed commits requirement enabled successfully!")
        else:
            print(
                f"Failed to enable signed commits requirement: {response.status_code}"
            )
            print(response.json())
    else:
        response = requests.delete(url, headers=headers)
        if response.status_code == 204:
            print("Signed commits requirement disabled successfully!")
        else:
            print(
                f"Failed to disable signed commits requirement: {response.status_code}"
            )
            print(response.json())
