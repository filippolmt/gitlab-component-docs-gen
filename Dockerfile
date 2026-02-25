FROM golang:1.26.0-alpine3.23 AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY main.go README.md.tmpl ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gitlab-component-docs-gen main.go

FROM scratch
COPY --from=builder /build/gitlab-component-docs-gen /gitlab-component-docs-gen
WORKDIR /app
ENTRYPOINT ["/gitlab-component-docs-gen"]
