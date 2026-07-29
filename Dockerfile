# syntax=docker/dockerfile:1

FROM node:24-alpine AS ui-builder

WORKDIR /src/ui

COPY ui/package.json ui/package-lock.json ./
RUN npm ci

COPY ui/ ./
RUN npm run generate

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download

COPY server/ ./server/
COPY --from=ui-builder /src/server/internal/static/dist ./server/internal/static/dist

ARG VERSION=dev
RUN cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/openlicensd \
    ./cmd/openlicensd

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/openlicensd /openlicensd

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/openlicensd"]
