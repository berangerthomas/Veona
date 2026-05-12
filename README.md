# Veona — System Monitoring Platform

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-007ACC?style=flat&logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![Docker](https://img.shields.io/badge/Docker-2CA5E0?style=flat&logo=docker&logoColor=white)](https://docker.com)

**Veona** is a monitoring platform that collects system metrics through a lightweight Go agent and sends them to a central server backed by VictoriaMetrics for time-series storage.

---

## Quick Start Guide

### Prerequisites

- **Docker Desktop** installed and running
- **Go** 1.22+ (to build the agent)
- **Node.js** 20+ (optional, for server development)

---

### 1. Start the server and VictoriaMetrics

```bash
cd deployment
docker-compose up -d --build
```

The API server is available at `http://localhost:3000` and VictoriaMetrics at `http://localhost:8428`.

> ✅ **Verification** : open `http://localhost/health` — the response should be `Veona API OK`.

---

### 2. Create a token

A test token is automatically created on first startup.  
To retrieve an existing probe token:

```bash
docker exec -it veona-server sqlite3 /app/data/veona.db "SELECT api_key FROM probes"
```

Copy the token — you will need it to configure the agent.

---

### 3. Build and run the agent

**From Windows (PowerShell) :**

```powershell
cd agent
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o veona-agent ./cmd/veona-agent/main.go
```

**From Linux :**

```bash
cd agent
GOOS=linux GOARCH=amd64 go build -o veona-agent ./cmd/veona-agent/main.go
```

---

### 4. Configure the agent

Create a `config.yaml` file:

```yaml
server:
  url: "http://localhost:3000/api/metrics"
  token: "<YOUR_TOKEN>"

buffer:
  size: 5000

collectors:
  cpu:     { enabled: true,  interval: "1m" }
  mem:     { enabled: true,  interval: "30s" }
  disk:    { enabled: true,  interval: "10m", auto_discover: true, exclude_fs: ["tmpfs","devtmpfs","squashfs","iso9660"] }
  net:     { enabled: true,  interval: "30s" }
  swap:    { enabled: true,  interval: "1m" }
  load:    { enabled: false, interval: "1m" }
  process_states: { enabled: false, interval: "1m" }
  temperatures:   { enabled: false, interval: "1m" }
  gpu:     { enabled: false, interval: "1m" }
  battery: { enabled: false, interval: "1m" }
  entropy: { enabled: false, interval: "1m" }
  time_sync: { enabled: false, interval: "2m" }
```

Run the agent:

```bash
./veona-agent --config ./config.yaml
```

---

## 📊 Querying metrics

Open VictoriaMetrics UI: `http://localhost/vmui/`

All metrics are prefixed with `veona_` and tagged with `hostname` and `probe_id`.

### CPU / Load metrics

```promql
veona_cpu_usage_percent
veona_cpu_core_count
veona_load_1
veona_load_5
veona_load_15
```

### Memory / Swap metrics

```promql
veona_mem_total
veona_mem_free
veona_mem_available
veona_mem_used_percent
veona_swap_total
veona_swap_free
veona_swap_used_percent
```

### Disk metrics

```promql
veona_disk_total{mountpoint="/"}
veona_disk_free{mountpoint="/"}
veona_disk_used_percent{hostname="my-server"}
```

### Network metrics

```promql
veona_net_bytes_recv
veona_net_bytes_sent
```

### Process metrics

```promql
veona_process_count_running
veona_process_count_sleeping
veona_process_count_stopped
veona_process_count_zombie
veona_process_count_idle
veona_process_count_other
veona_process_count_total
```

### GPU metrics (NVIDIA)

```promql
veona_gpu_0_utilization_percent
veona_gpu_0_mem_utilization_percent
veona_gpu_0_mem_used_mb
veona_gpu_0_mem_total_mb
```

### System metrics

```promql
veona_battery_capacity_percent
veona_system_entropy
veona_ntp_drift_ms
veona_agent_mem_alloc_bytes
veona_agent_mem_sys_bytes
veona_agent_goroutines
```

### Filtering by server

```promql
veona_cpu_usage_percent{hostname="prod-web-01"}
veona_mem_used_percent{hostname=~"prod-.*"}
```

---

## 🧱 Architecture

```
Go Agent (collect) → HTTP GZIP → Hono Server (validate) → VictoriaMetrics (store)
```

- **Agent**: collects CPU, memory, disk, network, GPU, etc. via `gopsutil`
- **Server**: receives metrics, validates tokens, transforms to Prometheus format
- **VictoriaMetrics**: time-series database compatible with PromQL

---

## 📁 Project structure

```
Veona/
├── agent/          # Go agent (metric collector)
│   ├── cmd/        # Entry point
│   └── internal/   # Buffer, collectors, HTTP dispatcher
├── deployment/     # Docker Compose, Nginx, Grafana
├── server/         # TypeScript/Hono server
│   └── src/        # API, validation, transformation
└── LICENSE
```

---

## 🔧 Useful commands

```bash
# Server logs
docker-compose -f deployment/docker-compose.yml logs -f

# Build the agent for Linux
cd agent && GOOS=linux GOARCH=amd64 go build -o veona-agent ./cmd/veona-agent/main.go

# Build the agent for Windows
cd agent && $env:GOOS="windows"; go build -o veona-agent.exe ./cmd/veona-agent/main.go

# Build the server
cd server && npm run build
```

---

## 📄 License

MIT — see the [LICENSE](LICENSE) file.