FROM --platform=$BUILDPLATFORM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS go-toolchain

FROM --platform=$BUILDPLATFORM node:24.22.2-bookworm-slim@sha256:01d566e8d94d8c619591d78652babfccd3c6992a3dffa9ed5ab39e11cf6c70e9 AS build
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=go-toolchain /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}" CGO_ENABLED=0 GOFLAGS=-trimpath
WORKDIR /workspace
COPY package.json package-lock.json ./
COPY apps/web/package.json ./apps/web/
COPY packages/contracts/package.json ./packages/contracts/
RUN npm ci
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
RUN cd apps/api && go mod download
COPY apps/api/ ./apps/api/
COPY apps/web/ ./apps/web/
COPY packages/contracts/ ./packages/contracts/
COPY scripts/generate-provider-icons.mjs ./scripts/
COPY scripts/provider-sources/ ./scripts/provider-sources/
COPY .prettierrc.json ./
ARG TARGETOS
ARG TARGETARCH
RUN npm run contracts:check \
    && npm run providers:check \
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} npm run build

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /workspace/dist/bank-piggy /app/bank-piggy
COPY --from=build /workspace/apps/web/dist/ /app/web/
WORKDIR /app
USER 65532:65532
ENV APP_MODE=private PORT=8080 LISTEN_ADDR=0.0.0.0:8080 WEB_DIST_DIR=/app/web
EXPOSE 8080
STOPSIGNAL SIGTERM
ENTRYPOINT ["/app/bank-piggy"]
