FROM golang:1.25-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /irtrafficgen .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /irtrafficgen /irtrafficgen
ENTRYPOINT ["/irtrafficgen", "run", "--config", "/etc/irtrafficgen/config.yaml", "--live"]
