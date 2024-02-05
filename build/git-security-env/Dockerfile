FROM python:3.12.1-alpine3.19

RUN apk --no-cache add curl jq

RUN python -m pip install requests

COPY *.sh /
COPY *.py /
