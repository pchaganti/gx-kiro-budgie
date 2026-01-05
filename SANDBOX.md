# Sandbox Feature Specification

This document describes the `--sandbox` feature for Budgie, which runs sub-agents inside isolated Docker containers.

## Motivation

Running AI agents with full access to the host filesystem poses security risks:
- Accidental deletion of files (documented cases with Claude Code, Google Antigravity)
- Credential leakage from `~/.aws/`, `~/.ssh/`, etc.
- Uncontrolled execution of shell commands

Docker containers provide process and filesystem isolation, limiting the blast radius of agent actions.

## CLI Interface

```bash
./budgie --sandbox [--sandbox-image image-name]
```

- `--sandbox` enables containerized execution (boolean flag)
- `--sandbox-image` specifies the Docker image (default: `budgie-sandbox:latest`)
- Image must have Linux kiro-cli pre-installed

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Host (macOS/Linux)                       │
│                                                                  │
│  ┌──────────────┐                                               │
│  │   Budgie     │                                               │
│  │  MCP Server  │                                               │
│  └──────┬───────┘                                               │
│         │                                                        │
│         │ docker run                                             │
│         ▼                                                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Docker Container                            │    │
│  │                                                          │    │
│  │  /workspace        ← input.Directory (RW)               │    │
│  │  /root/.local/share/kiro-cli  ← session volume (RW)     │    │
│  │  /auth             ← host kiro auth (RO)                │    │
│  │  /root/.kiro/      ← agent configs (RO)                 │    │
│  │                                                          │    │
│  │  kiro-cli chat --agent <name> --no-interactive <prompt> │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  Docker Volumes:                                                 │
│  budgie-session-<uuid> ← one per session, deleted on exit       │
└─────────────────────────────────────────────────────────────────┘
```

## Container Mounts

| Source (Host) | Container Path | Mode | Purpose |
|---------------|----------------|------|---------|
| `input.Directory` | `/workspace` | RW | User's working directory |
| `budgie-session-<sessionId>` (Docker volume) | `/root/.local/share/kiro-cli` | RW | kiro-cli SQLite + session state |
| `~/Library/Application Support/kiro-cli/` (macOS) or `~/.local/share/kiro-cli/` (Linux) | `/auth` | RO | Auth tokens source |
| `~/.kiro/` | `/root/.kiro/` | RO | All kiro configs (agents, settings, prompts, sub-agents) |

## Key Design Decisions

### 1. One Docker Volume Per Session

**Decision**: Create a separate Docker volume for each sessionId.

**Alternatives Considered**:
- **Shared volume for all sessions**: Simpler volume management, but SQLite concurrent write contention risk
- **Host directory mounts**: No volume management, but less isolation

**Why chosen**: 
- Eliminates SQLite contention entirely
- Each session is fully isolated
- Clean cleanup on budgie exit
- Matches existing session isolation model (one directory per session)

**Note**: During discussion, we confirmed that multiple kiro-cli instances on macOS work fine with shared SQLite (WAL mode handles it). However, separate volumes reduce risk and maintain cleaner isolation.

### 2. Auth Token Handling

**Decision**: Mount host's kiro-cli data directory read-only at `/auth`, copy `data.sqlite3` to session volume on container start.

**Alternatives Considered**:
- **Mount auth RW**: Risk of container corrupting host auth data
- **Re-authenticate in each container**: Poor UX, requires interactive login
- **Separate auth-only SQLite**: kiro-cli doesn't support split storage

**Why chosen**:
- Single login on host works for all containers
- Read-only mount protects host data
- Copy operation is fast and simple
- Container has writable location for conversation history

**Implementation**:
```bash
sh -c "cp /auth/data.sqlite3 /root/.local/share/kiro-cli/ 2>/dev/null; kiro-cli ..."
```

### 3. Prompt Enhancement for Container Paths

**Decision**: Change working directory in prompt from host path to `/workspace`.

**Current behavior** (non-sandbox):
```
"In directory /Users/x/project, <prompt>"
```

**Sandbox behavior**:
```
"In directory /workspace, <prompt>"
```

**Why**: The container doesn't see host paths. The working directory is mounted at `/workspace`.

### 4. Session Tracking and Cleanup

**Decision**: Unified `sessions map[string]bool` in `sessions.Manager`, with mode-specific cleanup.

**Current structure**:
```go
type Manager struct {
    baseDir     string
    createdDirs map[string]bool  // tracks directories
    dirsMutex   sync.Mutex
}
```

**New structure**:
```go
type Manager struct {
    baseDir     string
    sessions    map[string]bool  // tracks sessionIds (both modes)
    mutex       sync.Mutex
    sandboxMode bool
}
```

**Cleanup behavior**:
| Mode | Cleanup Action |
|------|----------------|
| Normal | `os.RemoveAll(filepath.Join(baseDir, sessionId))` |
| Sandbox | `docker volume rm budgie-session-<sessionId>` |

**Important**: Each budgie instance only cleans up sessions it created. Never touch other instances' volumes.

### 5. Why Not Use Docker's Built-in Sandboxes?

Docker Desktop 4.50+ has `docker sandbox run` (experimental). We chose custom implementation because:

- **Portability**: Works on Linux servers without Docker Desktop
- **Control**: Full control over volume management and lifecycle
- **No experimental features**: Production-ready without feature flags
- **Flexibility**: Can customize isolation level, mounts, networking

### 6. Cross-Platform Considerations

**macOS host → Linux container**:
- kiro-cli binary must be Linux (cannot mount macOS binary)
- SQLite format is cross-platform (no issues)
- Auth data path differs: macOS `~/Library/Application Support/kiro-cli/` vs Linux `~/.local/share/kiro-cli/`

**Solution**: Budgie detects host OS and mounts from correct path.

## kiro-cli Session Management

Understanding how kiro-cli stores context is critical for `--resume` to work.

### Storage Location
- **macOS**: `~/Library/Application Support/kiro-cli/data.sqlite3`
- **Linux**: `~/.local/share/kiro-cli/data.sqlite3`

### Database Schema (relevant tables)
```sql
CREATE TABLE conversations_v2 (
    key TEXT NOT NULL,           -- working directory path
    conversation_id TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (key, conversation_id)
);

CREATE TABLE auth_kv (
    key TEXT PRIMARY KEY,
    value TEXT
);
```

### How `--resume` Works
- kiro-cli uses `pwd` (current working directory) as the `key` in `conversations_v2`
- `--resume` flag loads conversation history where `key = $(pwd)`
- In container: `key = /root/.local/share/kiro-cli` (the session volume mount point)

### Why This Works
- Each session volume is mounted at the same container path
- kiro-cli writes to that path, creating consistent keys
- Subsequent calls with same sessionId mount same volume → `--resume` finds history

## Docker Image

A custom image is required because:
1. kiro-cli must be Linux binary (can't mount macOS binary)
2. Need consistent tool environment

### Dockerfile

```dockerfile
FROM buildpack-deps:bookworm

# OpenJDK 21
RUN apt-get update && apt-get install -y --no-install-recommends openjdk-21-jdk \
    && rm -rf /var/lib/apt/lists/*

# kubectl
RUN curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | gpg --dearmor -o /usr/share/keyrings/kubernetes.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/kubernetes.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /" > /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update && apt-get install -y --no-install-recommends kubectl \
    && rm -rf /var/lib/apt/lists/*

# Additional dev tools
RUN apt-get update && apt-get install -y --no-install-recommends jq tree ripgrep \
    && rm -rf /var/lib/apt/lists/*

# kiro-cli
RUN curl -fsSL https://kiro.dev/install.sh | sh

# sqlite3 for entrypoint auth sync
RUN apt-get update && apt-get install -y --no-install-recommends sqlite3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
```

### Entrypoint Script (`docker/entrypoint.sh`)

```bash
#!/bin/sh
# Budgie sandbox entrypoint
# Copies auth tokens from host while preserving session conversation history

KIRO_DATA_DIR="/root/.local/share/kiro-cli"
AUTH_SOURCE="/auth/data.sqlite3"
TARGET_DB="$KIRO_DATA_DIR/data.sqlite3"

mkdir -p "$KIRO_DATA_DIR"

if [ ! -f "$TARGET_DB" ]; then
    # First run: copy entire database (includes auth + empty conversations)
    cp "$AUTH_SOURCE" "$TARGET_DB" 2>/dev/null
elif [ -f "$AUTH_SOURCE" ]; then
    # Subsequent runs: only update auth tokens, preserve conversations
    sqlite3 "$TARGET_DB" "ATTACH '$AUTH_SOURCE' AS auth_src; \
        DELETE FROM auth_kv; \
        INSERT INTO auth_kv SELECT * FROM auth_src.auth_kv;" 2>/dev/null
fi

exec "$@"
```

**Why this approach**:
- First container run: copies full database (auth + schema)
- Subsequent runs: only syncs `auth_kv` table, preserving `conversations_v2`
- Ensures `--resume` works across container restarts

### Build Command
```bash
docker build -t budgie-agent:latest .
```

## Full Docker Run Command

```bash
docker run --rm \
  -v "/host/working/dir:/workspace:rw" \
  -v "budgie-session-<sessionId>:/root/.local/share/kiro-cli:rw" \
  -v "$HOME/Library/Application Support/kiro-cli:/auth:ro" \
  -v "$HOME/.kiro:/root/.kiro:ro" \
  budgie-agent:latest \
  kiro-cli chat --agent <name> --no-interactive [--resume] '<prompt>'
```

Note: The entrypoint handles copying auth tokens automatically before executing the command.

## Implementation Tasks

### 1. CLI Flag Parsing
- [ ] Add `--sandbox` flag (optional string, defaults to `budgie-agent:latest`)
- [ ] Store sandbox mode and image name in `config.Config`

### 2. Modify `sessions.Manager`
- [ ] Rename `createdDirs` to `sessions`
- [ ] Add `sandboxMode bool` field
- [ ] Update `GetWorkspaceDir()`:
  - Normal mode: create directory (existing behavior)
  - Sandbox mode: `docker volume create budgie-session-<sessionId>`
- [ ] Update `Cleanup()`:
  - Normal mode: `os.RemoveAll()` (existing behavior)
  - Sandbox mode: `docker volume rm budgie-session-<sessionId>`

### 3. Create Docker Executor
- [ ] New file: `internal/docker/executor.go`
- [ ] Function to build `docker run` command with all mounts
- [ ] Detect host OS for correct auth path:
  - macOS: `~/Library/Application Support/kiro-cli/`
  - Linux: `~/.local/share/kiro-cli/`
- [ ] Handle volume creation before first run

### 4. Modify `kiro.Executor`
- [ ] Add `sandboxMode` and `sandboxImage` fields
- [ ] In `executeOnce()`:
  - Normal mode: `exec.Command("kiro-cli", ...)` (existing)
  - Sandbox mode: `exec.Command("docker", "run", ...)` with all mounts

### 5. Modify Prompt Enhancement
- [ ] In `createHandler()` (main.go):
  - Normal mode: `"In directory {input.Directory}, {prompt}"`
  - Sandbox mode: `"In directory /workspace, {prompt}"`

### 6. Response File Handling
- [ ] Response file path changes in sandbox mode:
  - Normal: `{sessionDir}/response-{uuid}.txt`
  - Sandbox: Read from volume after container exits
- [ ] May need to mount response file location or copy out after execution

### 7. Update Makefile
- [ ] Add `docker-build` target to build the sandbox image
- [ ] Update `install` target to optionally build image

### 8. Documentation
- [ ] Update README.md with `--sandbox` usage
- [ ] Document Docker image requirements
- [ ] Document mount points and their purposes

## Edge Cases

### 1. Orphaned Volumes on Crash
- Volumes persist if budgie crashes before cleanup
- Recovery: `docker volume ls -q | grep budgie-session- | xargs docker volume rm`
- Consider: startup cleanup of old volumes (with caution - don't delete other instances' volumes)

### 2. Working Directory Validation
- Should validate `input.Directory` is an absolute path
- Consider: allowlist of permitted base paths for security

### 3. Docker Not Available
- Check for Docker availability on startup when `--sandbox` is used
- Fail fast with clear error message

### 4. Network Access
- Containers need outbound HTTPS for kiro-cli API calls
- Default Docker networking (`bridge`) should work
- Future consideration: restrict to specific endpoints

### 5. MCP Servers in Container
- MCP servers defined in `~/.kiro/settings/mcp.json` may reference host paths
- May need path translation or separate container MCP config
- For now: document limitation

## Testing Plan

1. **Unit tests**: Mock Docker commands, verify correct mount arguments
2. **Integration test**: Run simple agent task in sandbox, verify isolation
3. **Resume test**: Multiple calls with same sessionId, verify context persists
4. **Cleanup test**: Verify volumes deleted on graceful shutdown
5. **Cross-platform**: Test on macOS and Linux hosts

## Future Enhancements

1. **Network boundary**: Restrict outbound connections to allowlisted hosts
2. **Resource limits**: CPU/memory limits via `--cpus`, `--memory`
3. **Custom mounts**: Allow additional mounts via config
4. **Pre-built images**: Publish official images to Docker Hub/ECR
