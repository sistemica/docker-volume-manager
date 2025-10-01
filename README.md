# Sistemica Docker Volume Manager

A modern, pluggable volume management system for Docker Swarm with embedded etcd, REST API, and CSI driver support.

## Overview

The Docker Volume Manager provides cluster-aware storage management for Docker Swarm with a clean separation between control plane (management) and data plane (actual storage operations).

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│              Volume Manager (Control Plane)              │
│  • REST API for volume management                        │
│  • Embedded etcd for distributed metadata                │
│  • Pluggable storage backends                            │
│  • Multi-tenancy & RBAC (future)                         │
└────────────────────┬────────────────────────────────────┘
                     │ HTTP REST API
┌────────────────────▼────────────────────────────────────┐
│            CSI Driver Plugin (Data Plane)                │
│  • NodeStageVolume / NodePublishVolume                   │
│  • Host filesystem access with CAP_SYS_ADMIN             │
│  • Bind mount operations                                 │
└──────────────────────────────────────────────────────────┘
```

### Features

- **Pluggable Storage Backends**: Start with local filesystem, extend to distributed storage
- **Embedded etcd**: Automatic clustering, no external dependencies
- **REST API**: Clean HTTP API for volume management
- **CSI Driver**: Native Docker Swarm CSI plugin support
- **High Availability**: Deploy 3+ replicas for automatic failover
- **Zero Configuration**: Auto-discovery via Docker Swarm DNS

## Quick Start

### Prerequisites

- Docker 23.0+ (for CSI support)
- Docker Swarm initialized
- Go 1.21+ (for development)

### Development

```bash
# Clone repository
git clone https://github.com/sistemica/docker-volume-manager.git
cd docker-volume-manager

# Copy environment file
cp .env.example .env

# Install dependencies
go mod download

# Run locally
make run

# Run tests
make test

# Build binary
make build
```

### Production Deployment

```bash
# Step 1: Deploy Volume Manager service
docker stack deploy -c deploy/swarm-stack.yml volume-manager

# Step 2: Wait for API to be ready
curl http://localhost:9789/health

# Step 3: Install CSI plugin on all nodes
docker plugin install \
  --grant-all-permissions \
  sistemica/docker-volume-manager-csi:latest \
  MANAGER_URL=http://volume-manager:9789

# Step 4: Create a volume
docker volume create \
  --driver sistemica/docker-volume-manager-csi \
  --type mount \
  --opt backend=local \
  --opt path=/data/volumes/myvolume \
  myvolume

# Step 5: Use in a service
docker service create \
  --name web \
  --mount type=cluster,source=myvolume,target=/data \
  nginx:latest
```

## Storage Backends

### Current Backends

#### Local Filesystem
Simple local directory storage (phase 1).

```bash
docker volume create \
  --driver sistemica/docker-volume-manager-csi \
  --opt backend=local \
  --opt path=/mnt/data/myvolume \
  myvolume
```

### Planned Backends

#### Zip Archive (Phase 2)
Mount zip files as read-only volumes.

```bash
docker volume create \
  --driver sistemica/docker-volume-manager-csi \
  --opt backend=zip \
  --opt source=https://example.com/data.zip \
  --opt checksum=sha256:abc123... \
  myvolume
```

#### Distributed Storage (Phase 3)
Replicated storage with redundancy (Longhorn-inspired).

```bash
docker volume create \
  --driver sistemica/docker-volume-manager-csi \
  --opt backend=distributed \
  --opt replicas=3 \
  --opt size=10Gi \
  myvolume
```

## REST API

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |
| `POST` | `/api/v1/volumes` | Create volume |
| `GET` | `/api/v1/volumes` | List volumes |
| `GET` | `/api/v1/volumes/{id}` | Get volume details |
| `DELETE` | `/api/v1/volumes/{id}` | Delete volume |
| `POST` | `/api/v1/volumes/{id}/stage` | Stage volume on node |
| `DELETE` | `/api/v1/volumes/{id}/stage` | Unstage volume |
| `POST` | `/api/v1/volumes/{id}/publish` | Publish (mount) volume |
| `DELETE` | `/api/v1/volumes/{id}/publish` | Unpublish (unmount) volume |
| `GET` | `/api/v1/backends` | List available backends |

### Example: Create Volume

```bash
curl -X POST http://localhost:9789/api/v1/volumes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "myvolume",
    "backend": "local",
    "parameters": {
      "path": "/data/volumes/myvolume"
    }
  }'
```

### Example: File Operations (RESTful)

```bash
# Create/Update file
curl -X PUT http://localhost:9789/api/v1/volumes/{id}/files/data/config.json \
  -H "Content-Type: application/json" \
  -d '{"content":"{\"app\":\"test\"}"}'

# Read file
curl http://localhost:9789/api/v1/volumes/{id}/files/data/config.json

# List directory
curl http://localhost:9789/api/v1/volumes/{id}/files/data

# Delete file
curl -X DELETE http://localhost:9789/api/v1/volumes/{id}/files/data/config.json
```

Response:
```json
{
  "id": "vol-abc123",
  "name": "myvolume",
  "backend": "local",
  "parameters": {
    "path": "/data/volumes/myvolume"
  },
  "status": "created",
  "created_at": "2025-10-01T10:00:00Z"
}
```

## Project Structure

```
docker-volume-manager/
├── cmd/
│   ├── volume-manager/      # Control plane service
│   │   └── main.go
│   ├── csi-plugin/          # CSI driver plugin (future)
│   └── vmctl/               # CLI tool (future)
├── pkg/
│   ├── api/                 # HTTP API layer
│   │   ├── server.go
│   │   ├── handlers/
│   │   │   ├── volumes.go
│   │   │   ├── backends.go
│   │   │   └── health.go
│   │   └── middleware/
│   │       ├── logger.go
│   │       ├── recovery.go
│   │       └── cors.go
│   ├── storage/             # Storage backend interface
│   │   ├── backend.go       # Interface definition
│   │   ├── registry.go      # Backend registry
│   │   ├── local/           # Local filesystem backend
│   │   │   └── backend.go
│   │   └── mock/            # Mock for testing
│   ├── store/               # Metadata store (etcd)
│   │   ├── store.go         # Store interface
│   │   ├── etcd.go          # Embedded etcd (future)
│   │   └── memory.go        # In-memory (development)
│   ├── config/              # Configuration
│   │   └── config.go
│   └── types/               # Shared types
│       └── types.go
├── deploy/
│   ├── swarm-stack.yml      # Production deployment
│   └── docker-compose.yml   # Local development
├── .env.example             # Environment variables template
├── Makefile                 # Build automation
├── go.mod
├── go.sum
└── README.md
```

## Configuration

Environment variables (`.env`):

```bash
# Server Configuration
PORT=9789
HOST=0.0.0.0
LOG_LEVEL=info
ENVIRONMENT=development

# Storage Configuration
DATA_DIR=/var/lib/volume-manager

# Etcd Configuration
ETCD_ENABLED=true          # Enable embedded etcd
CLUSTER_SIZE=1             # Single node or cluster size
ETCD_CLIENT_PORT=2379
ETCD_PEER_PORT=2380

# Swarm Discovery
SERVICE_NAME=volume-manager
TASK_SLOT=1
```

## Development

### Architecture Principles

1. **Clean Architecture**: Clear separation of concerns (API → Service → Storage)
2. **Pluggable Backends**: Factory pattern with registry
3. **Interface-Driven**: All components depend on interfaces
4. **Dependency Injection**: Explicit dependencies
5. **Testable**: Unit tests with mocks

### Adding a New Backend

```go
// 1. Implement the Backend interface
type MyBackend struct {
    // ...
}

func (b *MyBackend) Stage(ctx context.Context, req StageRequest) error {
    // Implementation
}

func (b *MyBackend) Publish(ctx context.Context, req PublishRequest) error {
    // Implementation
}

// 2. Register in init()
func init() {
    storage.RegisterBackend("mybackend", NewMyBackend)
}

// 3. Import in main.go
import _ "github.com/sistemica/docker-volume-manager/pkg/storage/mybackend"
```

## Testing

```bash
# Unit tests
make test

# Integration tests
make test-integration

# Coverage
make coverage

# Lint
make lint
```

## Roadmap

### Phase 1: Core Infrastructure ✅ COMPLETE
- [x] Project structure
- [x] REST API with Echo
- [x] Local filesystem backend
- [x] Embedded etcd with persistence
- [x] RESTful file operations (files as resources)
- [x] Health checks
- [x] Structured logging (slog)
- [x] Configuration with .env

### Phase 2: CSI Driver (In Progress)
- [ ] CSI gRPC service implementation
- [ ] Docker managed plugin packaging
- [ ] Integration with embedded etcd
- [ ] Swarm cluster volume support
- [ ] End-to-end testing

### Phase 3: Advanced Backends
- [ ] Zip archive backend
- [ ] HTTP/S3 remote sources
- [ ] Caching layer
- [ ] Prometheus metrics

### Phase 4: Distributed Storage
- [ ] Replication backend
- [ ] Automatic failover
- [ ] Volume migration
- [ ] Health monitoring

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

Apache 2.0

## Authors

**Sistemica** - [GitHub](https://github.com/sistemica)
