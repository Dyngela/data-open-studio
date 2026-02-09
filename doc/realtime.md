# Real-Time Progress System

When a job executes, progress flows from the generated Go program to the browser in real-time.

## Data Flow

```
Generated Go Program
    |
    | NATS publish (tenant.TID.job.JID.progress)
    v
NATS Server (port 4222)
    |
    | NATS subscribe (tenant.TID.job.*.progress)
    v
NATSBridge (Go backend)
    |
    | hub.broadcast(jobID, payload)
    v
WebSocket Hub
    |
    | client.send channel
    v
WebSocket Client (browser)
    |
    | onProgress callback
    v
Angular Component (node status update)
```

## Components

### 1. Progress Reporter (`gen/lib/progress.go`)

Used by generated Go programs to report execution progress.

```go
type Progress struct {
    NodeID   int    `json:"nodeId"`
    NodeName string `json:"nodeName"`
    Status   Status `json:"status"`    // "running" | "completed" | "failed"
    RowCount int64  `json:"rowCount"`
    Message  string `json:"message"`
}

type ProgressFunc func(Progress)
```

**ProgressReporter**:
- Created with `NewProgressReporter(natsURL, tenantID, jobID)`
- If NATS connection fails: returns a no-op reporter (job continues without progress)
- Publishes to subject: `tenant.<tenantID>.job.<jobID>.progress`
- `ReportFunc()` returns a `ProgressFunc` callback
- `Close()` gracefully drains the NATS connection

Each node template calls `progress(Progress{...})` at intervals (e.g., every 1000 rows for DB input, every batch for DB output).

### 2. WebSocket Hub (`internal/realtime/hub.go`)

Central message routing for WebSocket connections.

```go
type Hub struct {
    clients       map[*Client]bool               // All connected clients
    subscriptions map[uint]map[*Client]bool       // jobID -> set of clients
    register      chan *Client                     // New client connected
    unregister    chan *Client                     // Client disconnected
    subscribe     chan subscribeMsg                // Client subscribes to job
    broadcast     chan broadcastMsg                // Message to broadcast
}
```

**Run()** - Main event loop (goroutine):
- `register`: Add client to clients map
- `unregister`: Remove client from all subscriptions, close send channel, delete from clients
- `subscribe`: Add client to `subscriptions[jobID]`
- `broadcast`: Send payload to all clients subscribed to `jobID`. If client buffer full (backpressure): disconnect it

### 3. WebSocket Client (`internal/realtime/client.go`)

Represents a single browser connection.

```go
type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan []byte        // Buffered outgoing message channel (256)
}
```

**Constants**:
- writeWait: 10s
- pongWait: 60s
- pingPeriod: 54s (90% of pongWait)
- maxMessageSize: 4096
- sendBufSize: 256

**ReadPump()** (goroutine per client):
- Reads JSON messages from WebSocket
- Handles `subscribe` action: `{ "action": "subscribe", "jobId": 123 }`
- Registers subscription with hub

**WritePump()** (goroutine per client):
- Sends messages from `client.send` channel to WebSocket
- Sends ping frames on ticker
- Handles write timeouts

### 4. WebSocket Auth (`internal/realtime/auth.go`)

**ServeWS(hub, jwtSecret, w, r)**:
- Extracts token from query parameter: `?token=<jwt>`
- Validates JWT with `pkg.ValidateToken(secret)`
- Upgrades HTTP to WebSocket (gorilla/websocket)
- Creates Client and registers with Hub
- Spawns ReadPump and WritePump goroutines

**Upgrader**: `CheckOrigin` allows all origins.

### 5. NATS Bridge (`internal/realtime/nats.go`)

Bridges NATS messages to the WebSocket hub.

```go
type NATSBridge struct {
    conn     *nats.Conn
    hub      *Hub
    tenantID string
}
```

**NewNATSBridge(natsURL, tenantID, hub)**: Connect to NATS.

**Subscribe()**:
- Subject pattern: `tenant.<tenantID>.job.*.progress` (wildcard for any job)
- On message:
  1. Parse jobID from subject (position 3 in dot-separated parts)
  2. Wrap in envelope: `{ "type": "job.progress", "jobId": <id>, "payload": <raw> }`
  3. Broadcast to hub for all subscribers of that jobID

**Close()**: Drain NATS connection.

### 6. Frontend WebSocket Client (`core/services/base-ws.service.ts`)

**JobRealtimeService**:
```typescript
class JobRealtimeService {
    state: Signal<'disconnected' | 'connecting' | 'connected'>

    subscribeToJob(jobId: number): Promise<void>
    disconnect(): void
    onProgress(listener: (event: ProgressEvent) => void): () => void  // Returns unsubscribe fn
}

interface ProgressEvent {
    jobId: number
    nodeId: number
    nodeName: string
    status: 'running' | 'completed' | 'failed'
    rowCount: number
    message: string
}
```

- Connects to `environment.wsUrl + '?token=' + jwt`
- Sends `{ action: 'subscribe', jobId: <id> }` after connection
- Parses incoming `job.progress` messages
- Emits `ProgressEvent` to registered listeners
- Auto-reconnects on disconnect (3 second delay)

### 7. Playground Integration

In the Playground component:
1. User clicks Execute
2. `JobService.execute(id)` sends POST to backend
3. Backend spawns generated Go program
4. Frontend calls `realtimeService.subscribeToJob(jobId)`
5. On progress events: `nodeGraphService.updateNodeStatus(nodeId, status)`
6. Node visual state updates (idle -> running -> success/error)

## Message Format

### NATS Message (raw payload)
```json
{
    "nodeId": 1,
    "nodeName": "DB Input",
    "status": "running",
    "rowCount": 5000,
    "message": "Processing rows..."
}
```

### WebSocket Message (wrapped by bridge)
```json
{
    "type": "job.progress",
    "jobId": 42,
    "payload": {
        "nodeId": 1,
        "nodeName": "DB Input",
        "status": "completed",
        "rowCount": 15000,
        "message": "15000 rows processed"
    }
}
```

## Ports

| Service | Port | Protocol |
|---------|------|----------|
| NATS Client | 4222 | TCP |
| NATS HTTP Monitor | 8222 | HTTP |
| WebSocket (Go) | 8081 | WS |
| API Server | 8080 | HTTP |
