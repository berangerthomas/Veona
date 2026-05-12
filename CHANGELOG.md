# Changelog

## [0.1.0] - 2026-05-12

Veona is a push-based system monitoring platform with a lightweight Go agent that collects CPU, memory, disk, network, GPU, and system metrics and ships them via HTTP/GZIP to a TypeScript/Hono server backed by VictoriaMetrics. Includes Docker Compose stack with Nginx reverse proxy and pre-configured Grafana.

### Added
- **Monorepo Architecture**: separation between Data Plane / Control Plane and Agent metrics.
- **Go Agent (Probe)**: 
  - metric collection via `gopsutil`.
  - Central YAML configuration management (`config.yaml`).
  - Goroutine multi-threading separating CPU, Memory, and Disk checks natively.
  - Disk Auto-Discovery with filesystem-level exclusion policies (`exclude_fs`).
  - Resilient Ring Buffer data-structure gracefully pushing dropped metrics back locally awaiting network restore.
  - HTTP push dispatcher handling GZIP compression and Bearer Token authorizations.
- **TypeScript / Hono Control Plane**: 
  - `bun/node` POST ingest endpoint.
  - Drizzle / SQLite backend database dynamically validating API Bearer Tokens.
  - In-Memory LRU Cache eliminating Database I/O bottlenecks.
- **VictoriaMetrics Data Plane**: 
  - JSON-to-Prometheus layout converter on the fly.
- **Docker-Compose Deployment**: Production ready deployment containing both the TS Server and VictoriaMetrics Time Series database.
