ARG GO_BUILDER_VERSION
ARG NODE_BUILDER_VERSION
ARG ALPINE_VERSION

FROM golang:${GO_BUILDER_VERSION} AS go-builder
COPY ./ /git-security
RUN cd /git-security && CGO_ENABLED=0 GOOS=linux make clean go

FROM node:${NODE_BUILDER_VERSION} AS node-builder
ARG BUILD_VERSION=1.0.0
RUN apk update && apk add make
COPY ./ /git-security
RUN cd /git-security && BUILD_VERSION=$BUILD_VERSION make clean ui

FROM alpine:${ALPINE_VERSION}
COPY --from=go-builder /git-security/dist/git-security /
COPY --from=node-builder /git-security/dist/ui /ui
EXPOSE 8080
ENTRYPOINT [ "/git-security" ]
