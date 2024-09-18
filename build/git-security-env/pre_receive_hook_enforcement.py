#!/usr/local/bin/python

import json
import os
import requests
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

GITHUB_HOST = os.environ["GITHUB_HOST"]
GH_TOKEN = os.environ["GH_TOKEN"]
GIT_PRE_RECEIVE_HOOKS_ENABLE = os.getenv("GIT_PRE_RECEIVE_HOOKS_ENABLE", "")
GIT_PRE_RECEIVE_HOOKS_DISABLE = os.getenv("GIT_PRE_RECEIVE_HOOKS_DISABLE", "")
GIT_REPO_JSON = os.environ["GIT_REPO_JSON"]
repo = json.loads(GIT_REPO_JSON)

headers = {
    "Authorization": f"Bearer {GH_TOKEN}",
    "Accept": "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
}

def list_github_hooks():
    list_hooks_url = "https://%s/api/v3/repos/%s/pre-receive-hooks" % (
        GITHUB_HOST,
        repo["full_name"]
    )
    response = requests.get(list_hooks_url, headers=headers, verify=False)
    if response.status_code == 200:
        return response.json()
    else:
        print(f"Failed to retrieve hooks: {response.status_code} - {response.text}")
        return None

def update_hook_enforcement(hook_id, action):
    update_hooks_url = "https://%s/api/v3/repos/%s/pre-receive-hooks/%d" % (
        GITHUB_HOST,
        repo["full_name"],
        hook_id
    )
    data = {
        "enforcement": action
    }
    response = requests.patch(update_hooks_url, headers=headers, json=data, verify=False)
    if response.status_code == 200:
        print(f"Hook {action} successfully.")
    else:
        print(f"Failed to update hook enforcement: {response.status_code} - {response.text}")

def process_hooks(action, hooks_to_process):
    pre_receive_hooks = repo.get("customs", {}).get("pre-receive-hooks", [])
    hooks = list_github_hooks()

    if hooks:
        for hook_name in hooks_to_process:
            is_enabled = hook_name in pre_receive_hooks
            if (action == "enabled" and is_enabled) or (action == "disabled" and not is_enabled):
                print(f"The pre-receive hook {hook_name} is already {action}.")
            else:
                hook_id = next((hook['id'] for hook in hooks if hook['name'] == hook_name), None)
                if hook_id:
                    update_hook_enforcement(hook_id, action)
                else:
                    print(f"Hook {hook_name} not found in the repository.")
    else:
        print("No hooks found or failed to retrieve hooks from GitHub API.")

if GIT_PRE_RECEIVE_HOOKS_DISABLE:
    hooks_to_disable = [hook.strip() for hook in GIT_PRE_RECEIVE_HOOKS_DISABLE.split(",")]
    process_hooks("disabled", hooks_to_disable)
else:
    print("No hooks specified for disabling.")

if GIT_PRE_RECEIVE_HOOKS_ENABLE:
    hooks_to_enable = [hook.strip() for hook in GIT_PRE_RECEIVE_HOOKS_ENABLE.split(",")]
    process_hooks("enabled", hooks_to_enable)
else:
    print("No hooks specified for enabling.")