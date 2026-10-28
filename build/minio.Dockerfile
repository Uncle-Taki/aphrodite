# Build upstream source because the public MinIO container repositories no
# longer support fresh deployments. Keep releases explicit and reproducible.
FROM golang:1.23.6-bookworm AS client-build
ARG GOPROXY=https://proxy.golang.org|https://goproxy.cn|direct
ENV CGO_ENABLED=0 GOPROXY=${GOPROXY}
WORKDIR /src
RUN curl --fail --location --retry 3 --connect-timeout 20 --max-time 180 \
      https://codeload.github.com/minio/mc/tar.gz/refs/tags/RELEASE.2025-02-21T16-00-46Z \
      -o /tmp/source.tar.gz \
    && tar -xzf /tmp/source.tar.gz --strip-components=1
RUN go build -trimpath -o /out/mc .

FROM golang:1.23.6-bookworm AS server-build
ARG GOPROXY=https://proxy.golang.org|https://goproxy.cn|direct
ENV CGO_ENABLED=0 GOPROXY=${GOPROXY}
WORKDIR /src
RUN curl --fail --location --retry 3 --connect-timeout 20 --max-time 180 \
      https://codeload.github.com/minio/minio/tar.gz/refs/tags/RELEASE.2025-02-28T09-55-16Z \
      -o /tmp/source.tar.gz \
    && tar -xzf /tmp/source.tar.gz --strip-components=1
RUN go build -trimpath -o /out/minio .

FROM alpine:3.21 AS client
RUN apk add --no-cache ca-certificates
COPY --from=client-build /out/mc /usr/local/bin/mc
ENTRYPOINT ["mc"]

FROM alpine:3.21 AS server
RUN apk add --no-cache ca-certificates
COPY --from=server-build /out/minio /usr/local/bin/minio
EXPOSE 9000 9001
ENTRYPOINT ["minio"]
CMD ["server", "/data", "--console-address", ":9001"]
