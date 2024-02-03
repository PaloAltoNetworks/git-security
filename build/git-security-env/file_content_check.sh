#!/bin/sh

FILE=$(curl -skL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GH_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://$GITHUB_HOST/api/v3/repos/$GIT_REPO/contents/$FILE_PATH | \
    jq -r 'if .message then "RmlsZSBub3QgZXhpc3RlZAo=" else .content end' | \
    base64 -d)

if [ "$FILE" = "File not existed" ]; then
    echo ${OUTPUT_IF_NOT_FOUND:-"File doesn't exist"}
else
    if [ `echo $FILE | grep -c "$MATCH" ` -gt 0 ]; then
        echo ${OUTPUT_IF_MATCHED:-Enabled}
    else
        echo ${OUTPUT_IF_NOT_MATCHED:-Disabled}
    fi
fi

sleep ${DELAY:-1}
