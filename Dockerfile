FROM golang:1.24 AS builder

ARG KUSTOMIZE_VERSION=v5.0.3
ARG HELM_VERSION=v3.13.2
ARG GIT_VERSION=2.48.1

WORKDIR /go/src/github.com/OpsMx/argocd-policy-plugin/

COPY go.* ./
RUN go mod download

COPY . .
RUN ./deps.sh

RUN GOOS="linux" GOARCH="amd64" CGO_ENABLED=0 go build -o argocd-policy-plugin *.go

########################################
# Final argocd-policy-plugin stage
########################################

FROM alpine:3.18.4 AS argocd-policy-plugin

COPY --from=builder /usr/local/bin/helm /usr/local/bin/helm
COPY --from=builder /usr/local/bin/kustomize /usr/local/bin/kustomize
COPY --from=builder /go/src/github.com/OpsMx/argocd-policy-plugin/argocd-policy-plugin /usr/local/bin/argocd-policy-plugin

RUN apk update
RUN apk add git