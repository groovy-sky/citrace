FROM golang:1.25-alpine3.22 AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/citrace-shell ./cmd/citrace-shell

FROM busybox:1.37.0-musl AS runtime

FROM scratch
COPY --from=runtime /bin/busybox /bin/sh
COPY --from=build /out/citrace-shell /citrace-shell
USER 65532:65532
ENTRYPOINT ["/citrace-shell"]