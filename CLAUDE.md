# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All commands use the [Task](https://taskfile.dev) runner (`Taskfile.yaml`).

```bash
# Run the full dev stack in tmux (backend + control center + frontend)
task dev

# Run individual services
task backend      # OpenStack microservice (port 50052)
task control      # Control center gRPC+REST server (ports 50051, 50055)
task frontend     # SvelteKit dev server (port 80, requires sudo)
task frontexpose  # SvelteKit exposed on network (port 5173)
task auth         # Start GLAuth LDAP + Dex OIDC via Docker (ports 3893, 5556)

# Build
task build        # Build OpenStack microservice binary

# Setup
task setup        # Interactive configuration wizard
task init_front   # Install frontend npm dependencies

# Cleanup
task clean        # Remove build artifacts and SQLite DB
```

**Frontend only** (in `frontend/`):
```bash
npm run dev    # Dev server
npm run build  # Production build
npm run check  # TypeScript type check
```

## Architecture

Three-tier system with gRPC throughout:

```
SvelteKit Frontend
      │ gRPC-Web (/rpc/*)
      ▼
Control Center (Go)  ──── gRPC ────►  OpenStack Microservice (Go)
      │                                       │
      │ PostgreSQL LISTEN/NOTIFY              │ SQLite (job queue)
      ▼                                       ▼
  PostgreSQL                           OpenStack API
```

### Services

| Service | Path | Port | Role |
|---------|------|------|------|
| Control Center | `control_center/` | 50051 (gRPC), 50055 (REST) | Orchestration, users, pools, scheduling |
| OpenStack Microservice | `microservices/openstack/` | 50052 | VM provisioning via gophercloud |
| Frontend | `frontend/` | 5173 / 80 | SvelteKit UI |
| Caddy | `caddy/` | 443 | Reverse proxy — routes gRPC-Web + REST |
| Auth | `auth/` (gitignored) | 3893, 5556 | GLAuth (LDAP) + Dex (OIDC/PKCE) |

### Key Data Flows

**Pool creation:** Frontend → `CreatePool` RPC → Control Center stores in PostgreSQL → sends `RessourceRequest` to microservice → job queued in SQLite → worker calls OpenStack API → VMs register via REST `/api/register` → PostgreSQL NOTIFY streams status back to frontend.

**VM assignment:** `AttribVMinPool` RPC → allocates available VM from pool → injects student SSH key → returns IP + username.

**Real-time updates:** Control Center streams `UpdateDataUser` via PostgreSQL LISTEN/NOTIFY. Frontend subscribes and updates Svelte stores.

### Proto Definitions

Two proto files define all service contracts:
- `proto/frontcontrol.proto` — Frontend ↔ Control Center (AuthService, GatherDataService, PoolService, ConfigService, UserService, AttribVMService)
- `proto/poolmanager.proto` — Control Center ↔ Microservice (PoolManager: create/delete VMs, stream status, list resources)

Generated code: Go in `*/pb/` and `control_center/frontcontrolpb/`, TypeScript in `frontend/src/lib/grpc/`.

### Control Center internals (`control_center/internal/`)

Each subdirectory is a self-contained feature package:
- `pool/` — pool CRUD + cron scheduling
- `attribvm/` — VM-to-student attribution
- `monitoring/` — heartbeat tracking, activity detection
- `sshinject/` — post-boot SSH key injection into VMs
- `gatherdata/` — proxy to fetch images/flavors/networks from microservice
- `configpool/` — user-defined cloud-init scripts
- `oidc/` — JWT validation middleware, GLAuth integration
- `user/` — user management

### Frontend stores

Svelte stores in `frontend/src/lib/store/`:
- `authStore.ts` — auth state (token, email, role), OIDC PKCE flow, `loadAll` trigger
- `serverpoolStore.ts` — images, flavors, networks, servers, pools, configs; `loadAll()` fetches all on auth change

`loadAll` is called in the `authStore` subscriber (not just `onMount`) to handle post-OIDC-callback hydration.

### gRPC transport (frontend)

`frontend/src/lib/grpc/transport.ts` — authenticated gRPC-Web transport that reads `accessToken` from `authStore` and injects it as `Bearer` header. All service clients use this transport.

## Configuration

Copy `.env.example` to `.env` and configure:
- `POSTGRES_*` — central PostgreSQL connection
- `OS_CLOUD` / `INFRA_OS_CLOUD` — two OpenStack project names (VM creation vs. infra listing)
- `SSH_PUBLIC/PRIVATE_KEY_PATH` — SSH keys for student VM access
- `REGISTRAR_CONTROL_CENTER_URL` — URL VMs call back to on boot

OpenStack credentials go in `clouds.yaml` (from `clouds.yaml.template`).

Auth configs (`auth/dex.yaml`, `auth/glauth.cfg`) are gitignored — never commit them.

## Databases

| DB | Used by | Purpose |
|----|---------|---------|
| PostgreSQL | Control Center | Pools, servers, users, configs, VM registrar |
| SQLite (`PoolManagerVM.db`) | Microservice | Local job queue for async VM operations |

VM auto-registration schema: `sql/registrar_schema.sql`.
