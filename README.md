# ionos-cloud-watchdog

A diagnostic tool for IONOS Cloud and Kubernetes health checks.

## Installation

### Download binary

Download the latest release from [GitHub Releases](https://github.com/peterpisarcik/ionos-cloud-watchdog/releases).

```bash
# macOS (Apple Silicon)
curl -L https://github.com/peterpisarcik/ionos-cloud-watchdog/releases/latest/download/ionos-cloud-watchdog-darwin-arm64 -o ionos-cloud-watchdog
chmod +x ionos-cloud-watchdog

# macOS (Intel)
curl -L https://github.com/peterpisarcik/ionos-cloud-watchdog/releases/latest/download/ionos-cloud-watchdog-darwin-amd64 -o ionos-cloud-watchdog
chmod +x ionos-cloud-watchdog

# Linux
curl -L https://github.com/peterpisarcik/ionos-cloud-watchdog/releases/latest/download/ionos-cloud-watchdog-linux-amd64 -o ionos-cloud-watchdog
chmod +x ionos-cloud-watchdog

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/peterpisarcik/ionos-cloud-watchdog/releases/latest/download/ionos-cloud-watchdog-windows-amd64.exe -OutFile ionos-cloud-watchdog.exe
```

### Go install

```bash
go install github.com/peterpisarcik/ionos-cloud-watchdog/cmd/ionos-cloud-watchdog@latest
```

### Build from source

```bash
git clone https://github.com/peterpisarcik/ionos-cloud-watchdog.git
cd ionos-cloud-watchdog
go build ./cmd/ionos-cloud-watchdog
```

## Configuration

### Option 1: Configuration file (recommended)

Initialize a config file to store your credentials:

```bash
# Using flags (avoids terminal paste issues)
./ionos-cloud-watchdog config init --token "your-token-here"

# Or with username/password
./ionos-cloud-watchdog config init --username "user" --password "pass"

# Interactive mode (prompts for input)
./ionos-cloud-watchdog config init
```

This creates `~/.ionos-cloud-watchdog/config.yaml` with your credentials.

### Option 2: Environment variables

```bash
export IONOS_TOKEN=your-token-here
```

### Option 3: Command-line flags

```bash
# Set token via environment for this run only
IONOS_TOKEN=your-token ./ionos-cloud-watchdog
```

**Configuration priority:** config file < environment variables < command-line flags

## Usage

```bash
# Run the watchdog (uses config file or env vars)
./ionos-cloud-watchdog

# With custom kubeconfig
./ionos-cloud-watchdog --kubeconfig /path/to/kubeconfig

# Check specific namespace
./ionos-cloud-watchdog -n my-namespace

# JSON output
./ionos-cloud-watchdog -o json

# Verbose output
./ionos-cloud-watchdog -v

# Watch mode - refresh every 30 seconds
./ionos-cloud-watchdog -w 30
```

### Commands

```
config init          Initialize configuration file
completion           Generate shell completion scripts
help                 Help about any command
```

### Options

```
    --kubeconfig string   path to kubeconfig file
-n, --namespace string    kubernetes namespace to check (default: all)
-o, --output string       output format: text or json (default "text")
-v, --verbose             verbose output
-w, --watch int           watch mode: refresh interval in seconds (0 = disabled)
-h, --help                help for ionos-cloud-watchdog
```

### Environment Variables

- `IONOS_TOKEN` - IONOS Cloud API token ([how to generate](https://docs.ionos.com/cloud/set-up-ionos-cloud/management/identity-access-management/token-manager#generate-authentication-token))
- `IONOS_USERNAME` - IONOS Cloud username (alternative to token)
- `IONOS_PASSWORD` - IONOS Cloud password (alternative to token)

### Exit Codes

- `0` - OK
- `1` - WARNING (1-3 issues)
- `2` - CRITICAL (>3 issues)

## What it checks

**IONOS Cloud**
- Status page for outages
- API connectivity
- Authentication
- Datacenters with servers and volumes
- Kubernetes clusters and node pools
- Managed Databases (DBaaS)
  - PostgreSQL clusters
  - MongoDB clusters
  - MariaDB clusters
  - In-Memory DB instances

**Kubernetes**
- Node status and conditions (MemoryPressure, DiskPressure, PIDPressure)
- Pod status (CrashLoopBackOff, ImagePullBackOff, Pending, Failed)
- Deployment availability
- PVC binding status
- LoadBalancer services
- TLS certificate expiry (warns if < 30 days)

## Example Output

```
IONOS Cloud
-----------
  Status Page     OK
  API             OK
  Authentication  OK

Datacenters
-----------
  my-datacenter (de/txl)
    Servers: 4
    Volumes: 10
    State: OK

Kubernetes Clusters
-------------------
  my-cluster (v1.31.10)
    Node Pools: 2
    State: ACTIVE

Managed Databases
-----------------
  PostgreSQL: 2 cluster(s)
  MongoDB: 1 cluster(s)
  State: OK

Health
------
  Nodes          3/3 Ready
  Pods           45/45 Running
  Deployments    12/12 Available
  PVCs           8/8 Bound
  LoadBalancers  2/2 Ready
  Certificates   5/5 Valid

Status: OK
```

Use `--verbose` to see individual server and volume names.
