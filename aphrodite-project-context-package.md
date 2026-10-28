# PROJECT CONTEXT PACKAGE — APHRODITE

Generated engineering memory for continuation by another AI coding agent.

## Project Identity

Project Name:
Aphrodite

Purpose:
A Docker Compose based multi-service application containing:
- Go API
- Go Worker
- Strapi CMS
- PostgreSQL
- Redis
- MinIO
- imgproxy

## User Preferences

- Prefer complete Dockerfile/docker-compose replacements.
- Avoid partial instructions when changing deployment files.
- Explain root cause before fixes.
- Avoid destructive cleanup commands without verification.
- Optimize for small VPS environments.

## Current Status

Completed:
- Docker working.
- BuildKit working.
- Docker registry mirror configured.
- Docker build GC configured.
- VM cleaned.
- Swap configured.

Current VM:
- 15GB disk
- ~3.5GB used
- ~11GB free

## Docker Configuration

/etc/docker/daemon.json:

```json
{
  "registry-mirrors": [
    "https://docker.arvancloud.ir"
  ],
  "builder": {
    "gc": {
      "enabled": true,
      "defaultKeepStorage": "1GB"
    }
  }
}
```

## Current Blocking Issue

Strapi build fails:

```
RUN npm run build
```

Error:

```
FATAL ERROR:
Ineffective mark-compacts near heap limit
Allocation failed - JavaScript heap out of memory
```

Cause:
Node.js heap limit during Strapi admin panel compilation.

Recommended fix:

Add before npm build:

```dockerfile
ENV NODE_OPTIONS="--max-old-space-size=2048"
```

Also consider:

```dockerfile
RUN npm install --no-audit --no-fund
```

## Important Build Details

Go build commands:

```dockerfile
RUN go build -o /out/aphrodite-api ./cmd/api  && go build -o /out/aphrodite-worker ./cmd/worker
```

Do not replace with:

```
go build .
```

because the project has multiple binaries.

## Infrastructure History

### Go Proxy Issue

Problem:

```
proxy.golang.org 403 Forbidden
```

Solution:

Use:

```
GOPROXY=https://goproxy.cn,direct
```

### Docker Build Memory Issue

Problem:

```
failed to execute bake: signal: killed
```

Cause:
Parallel builds exhausted memory.

Solution:
Use:

```
COMPOSE_PARALLEL_LIMIT=1
```

and swap.

### Containerd Disk Issue

Problem:

/var/lib/containerd grew to ~8.3GB.

Solution:

Restart:

```bash
systemctl stop docker docker.socket
systemctl restart containerd
systemctl start docker
```

Result:
containerd reduced to ~2GB.

## Next Steps

1. Inspect Strapi Dockerfile.
2. Add NODE_OPTIONS heap increase.
3. Build Strapi separately:

```bash
docker compose -f build/docker-compose.yaml build strapi --progress=plain
```

4. Build remaining services.
5. Start stack:

```bash
docker compose -f build/docker-compose.yaml up -d
```

## Avoid

Do not:
- replace project Dockerfiles with generic examples.
- delete /var/lib/containerd manually.
- run docker prune with volumes.
- restore proxy.golang.org.
