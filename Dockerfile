FROM golang:1.19-alpine as build
ENV CGO_ENABLED=0
ENV GO111MODULE=auto
RUN apk add --no-cache make git
WORKDIR /src
RUN --mount=type=bind,source=.,rw \
  --mount=type=cache,target=/go/pkg/mod \
  go mod tidy && go mod download
RUN --mount=type=bind,source=.,rw \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
  go build -trimpath -ldflags "-s -w" -o /usr/local/onlooker main.go

FROM cgr.dev/chainguard/static
LABEL org.opencontainers.image.title=onlooker
LABEL org.opencontainers.image.base.name=cgr.dev/chainguard/static
COPY --from=build /usr/local/onlooker /usr/local/onlooker
ENTRYPOINT [ "/usr/local/onlooker" ]
