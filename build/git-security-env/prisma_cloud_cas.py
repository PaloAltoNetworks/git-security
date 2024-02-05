#!/usr/local/bin/python

import json
import os
import requests

# CATEGORY: SCA, SECRETS, IAC, or CICD
# SEVERITY: CRITICAL, HIGH, MEDIUM, LOW, INFO
categories = os.environ.get("CATEGORY", "SCA").split(",")
severties = os.environ.get("SEVERITY", "CRITICAL").split(",")

payload = {
    "username": os.environ["PRISMA_CLOUD_ACCESS_KEY"],
    "password": os.environ["PRISMA_CLOUD_SECRET_KEY"],
}
headers = {
    "Content-Type": "application/json; charset=UTF-8",
    "Accept": "application/json; charset=UTF-8",
}
response = requests.request(
    "POST",
    "{}/login".format(os.environ["PRISMA_CLOUD_API_URL"]),
    headers=headers,
    json=payload,
)

token = response.json()["token"]

headers = {
    "Content-Type": "application/json",
    "Accept": "application/json",
    "authorization": token,
}

results = {}
page = 1
while True:
    payload = {
        "filters": {},
        "pageConfig": {"page": page, "pageSize": 100},
    }

    r = requests.request(
        "POST",
        "{}/bridgecrew/api/v1/vcs-repository/repositories".format(
            os.environ["PRISMA_CLOUD_API_URL"]
        ),
        headers=headers,
        json=payload,
    )
    response = r.json()
    for repo in response:
        result = 0
        for c in categories:
            for s in severties:
                result += repo.get("issues", {}).get(c, {}).get(s, 0)
        results[repo["fullName"]] = result
    if len(response) < 100:
        break
    page += 1

print(json.dumps(results))
