FROM --platform=$BUILDPLATFORM golang:1.22.1-alpine3.19 AS builder
WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags '-w -s -extldflags "-static"' -tags time_tzdata -o /go/bin/app .

FROM scratch
COPY --from=builder /go/bin/app /app
COPY --from=alpine:3.19 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/app"]
EXPOSE 8080
