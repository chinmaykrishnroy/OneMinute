FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGET=./apps/server/cmd/server
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=locked CGO_ENABLED=0 GOMAXPROCS=2 go build -p 2 -trimpath -ldflags="-s -w" -o /out/service ${TARGET}

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
COPY --from=build /out/service /usr/local/bin/service
USER app
ENTRYPOINT ["/usr/local/bin/service"]
