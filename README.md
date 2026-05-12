# Veona — Système de Monitoring

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-007ACC?style=flat&logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![Docker](https://img.shields.io/badge/Docker-2CA5E0?style=flat&logo=docker&logoColor=white)](https://docker.com)

**Veona** est une plateforme de monitoring qui collecte des métriques systèmes via un agent Go léger et les envoie vers un serveur central qui les stocke dans VictoriaMetrics.

---

## 🚀 Guide de démarrage rapide

### Prérequis

- **Docker Desktop** installé et lancé
- **Go** 1.22+ (pour compiler l'agent)
- **Node.js** 20+ (optionnel, pour le développement du serveur)

---

### 1. Démarrer le serveur et VictoriaMetrics

```bash
cd deployment
docker-compose up -d --build
```

Le serveur API est disponible sur `http://localhost:3000` et VictoriaMetrics sur `http://localhost:8428`.

> ✅ **Vérification** : ouvrez `http://localhost:3000/health` — la réponse doit être `Veona API OK`.

---

### 2. Créer un token

Un token de test est automatiquement créé.  
Pour récupérer le token d'une sonde existante :

```bash
docker exec -it veona-server sqlite3 /app/data/veona.db "SELECT api_key FROM probes"
```

Copiez le token — vous en aurez besoin pour configurer l'agent.

---

### 3. Compiler et exécuter l'agent

**Depuis Windows (PowerShell) :**

```powershell
cd agent
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o veona-agent ./cmd/veona-agent/main.go
```

**Depuis Linux :**

```bash
cd agent
GOOS=linux GOARCH=amd64 go build -o veona-agent ./cmd/veona-agent/main.go
```

---

### 4. Configurer l'agent

Créez un fichier `config.yaml` :

```yaml
server:
  url: "http://localhost:3000/api/metrics"
  token: "<TOKEN_COPIÉ>"

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

Lancez l'agent :

```bash
./veona-agent --config ./config.yaml
```

---

## 📊 Visualiser les métriques

Ouvrez VictoriaMetrics UI : `http://localhost/vmui/`

Toutes les métriques sont préfixées par `veona_` et taguées avec `hostname` et `probe_id`.

### Métriques CPU / Charge

```promql
veona_cpu_usage_percent
veona_cpu_core_count
veona_load_1
veona_load_5
veona_load_15
```

### Métriques Mémoire / Swap

```promql
veona_mem_total
veona_mem_free
veona_mem_available
veona_mem_used_percent
veona_swap_total
veona_swap_free
veona_swap_used_percent
```

### Métriques Disque

```promql
veona_disk_total{mountpoint="/"}
veona_disk_free{mountpoint="/"}
veona_disk_used_percent{hostname="mon-serveur"}
```

### Métriques Réseau

```promql
veona_net_bytes_recv
veona_net_bytes_sent
```

### Métriques Processus

```promql
veona_process_count_running
veona_process_count_sleeping
veona_process_count_stopped
veona_process_count_zombie
veona_process_count_idle
veona_process_count_other
veona_process_count_total
```

### Métriques GPU (NVIDIA)

```promql
veona_gpu_0_utilization_percent
veona_gpu_0_mem_utilization_percent
veona_gpu_0_mem_used_mb
veona_gpu_0_mem_total_mb
```

### Métriques Système

```promql
veona_battery_capacity_percent
veona_system_entropy
veona_ntp_drift_ms
veona_agent_mem_alloc_bytes
veona_agent_mem_sys_bytes
veona_agent_goroutines
```

### Filtrer par serveur

```promql
veona_cpu_usage_percent{hostname="prod-web-01"}
veona_mem_used_percent{hostname=~"prod-.*"}
```

---

## 🧱 Architecture

```
Agent Go (collecte) → HTTP GZIP → Server Hono (validation) → VictoriaMetrics (stockage)
```

- **Agent** : collecte CPU, mémoire, disque, réseau, GPU, etc. via `gopsutil`
- **Server** : reçoit les métriques, valide le token, transforme au format Prometheus
- **VictoriaMetrics** : base de données time-series compatible PromQL

---

## 📁 Structure du projet

```
Veona/
├── agent/          # Agent Go (collecteur de métriques)
│   ├── cmd/        # Point d'entrée principal
│   └── internal/   # Buffer, collecteurs, dispatcher HTTP
├── deployment/     # Docker Compose, Nginx, Grafana
├── server/         # Serveur TypeScript/Hono
│   └── src/        # API, validation, transformation
└── LICENSE
```

---

## 🔧 Commandes utiles

```bash
# Logs du serveur
docker-compose -f deployment/docker-compose.yml logs -f

# Compiler l'agent pour Linux
cd agent && GOOS=linux GOARCH=amd64 go build -o veona-agent ./cmd/veona-agent/main.go

# Compiler l'agent pour Windows
cd agent && $env:GOOS="windows"; go build -o veona-agent.exe ./cmd/veona-agent/main.go

# Builder le serveur
cd server && npm run build
```

---

## 📄 Licence

MIT — voir le fichier [LICENSE](LICENSE).