#!/bin/sh

curl -skL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GH_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://$GITHUB_HOST/api/v3/repos/$GIT_REPO/pre-receive-hooks | \
    jq '.[] | select(.enforcement=="enabled") | .name' | \
    sort | \
    tr -d '"' | \
    jq -cRnM '[inputs]'

sleep ${DELAY:-1}
