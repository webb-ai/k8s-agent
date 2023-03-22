# syntax=docker/dockerfile:1.4

ARG BUILDER_IMAGE=golang:1.19.4-alpine3.17
ARG RUNTIME_IMAGE=alpine:3.17

FROM ${BUILDER_IMAGE} as go-builder

WORKDIR /build

# Copy and download dependency using go mod
COPY --link . .
RUN go mod download
RUN make clean check build

FROM ${RUNTIME_IMAGE}

WORKDIR /app

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY --from=go-builder ["/build/collector", "."]

ENTRYPOINT ["/app/collector"]
