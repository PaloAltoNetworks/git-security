#!/bin/sh

curl -skL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GH_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://$GITHUB_HOST/api/v3/repos/$GIT_REPO/contributors?anon=1 | \
    jq -r '.[] | if .login != null then .login else .email end' | \
    head -n 3 | sort | paste -sd "," -

sleep ${DELAY:-0.5}
