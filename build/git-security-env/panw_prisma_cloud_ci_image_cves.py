#!/usr/local/bin/python

# This script is specific to PANW use case
import base64
import os
import requests
import urllib3

urllib3.disable_warnings()

headers = {
    "Accept": "application/vnd.github+json",
    "Authorization": "Bearer {}".format(os.environ["GH_TOKEN"]),
    "X-GitHub-Api-Version": "2022-11-28",
}

r = requests.request(
    "GET",
    "https://{}/api/v3/repos/{}/contents/.harness/ci.yaml".format(
        os.environ["GITHUB_HOST"],
        os.environ["GIT_REPO"],
    ),
    headers=headers,
    verify=False,
)

response = r.json()
if "message" in response:
    print("")
    exit()

file_content = base64.b64decode(response["content"]).decode("utf-8")
git_repo_name = os.environ["GIT_REPO"].split("/")[1]
image_name = ""
for line in file_content.splitlines():
    if not line.strip().startswith("#") and " repo:" in line:
        line = line.replace(
            "<+<+codebase.repoUrl>.substring(<+codebase.repoUrl>.lastIndexOf('/') + 1)>",
            git_repo_name,
        )
        line = line.replace(
            "<+<+codebase.repoUrl>.substring(<+codebase.repoUrl>.lastIndexOf('/')",
            git_repo_name,
        )
        line = line.replace("repo:", "")
        line = line.strip()
        image_name = line
        break

if image_name != "":
    session = requests.Session()
    session.auth = (
        os.environ["PRISMA_CLOUD_USERNAME"],
        os.environ["PRISMA_CLOUD_PASSWORD"],
    )

    r = session.get(
        "{}/api/v1/scans?search={}&sort=time&reverse=true&limit=50".format(
            os.environ["CONSOLE_URL"],
            git_repo_name,
        ),
    )
    response = r.json()
    if response is None or len(response) == 0:
        print("")
        exit()

    image_repo_name = "/".join(image_name.split("/")[-2:])
    for scan in response:
        rt = scan["entityInfo"]["repoTag"]
        if rt["repo"] == image_repo_name and "pr" not in rt["tag"]:
            vd = scan["entityInfo"]["vulnerabilityDistribution"]
            print(vd["critical"] + vd["high"])
            exit()
