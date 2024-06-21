#!/bin/sh

# This script is specific to PANW use case

FILE=$(curl -skL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GH_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://$GITHUB_HOST/api/v3/repos/$GIT_REPO/contents/.harness/ci.yaml | \
    jq -r 'if .message then "RmlsZSBub3QgZXhpc3RlZAo=" else .content end')

if [ "$FILE" = "File not existed" ]; then
    echo ""
else
    PATTERN1="<+<+codebase.repoUrl>.substring(<+codebase.repoUrl>.lastIndexOf('/') + 1)>"
    PATTERN2="<+<+codebase.repoUrl>.substring(<+codebase.repoUrl>.lastIndexOf('/')"
    GIT_REPO_NAME=$(basename "$GIT_REPO")
    echo $FILE | \
        base64 -d | \
        grep -v '^\s*\#' | \
        grep "repo:" | \
        head -n 1 | \
        sed "s|$PATTERN1|$GIT_REPO_NAME|" | \
        sed "s|$PATTERN2|$GIT_REPO_NAME|" | \
        sed "s|repo:||" | \
        sed 's/^ *//;s/ *$//'
fi

sleep ${DELAY:-1}
