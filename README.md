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

## Usage

```bash
# Set your IONOS Cloud token
export IONOS_TOKEN=your-token-here

# Run the watchdog
./ionos-cloud-watchdog

# With custom kubeconfig
./ionos-cloud-watchdog --kubeconfig /path/to/kubeconfig

# Check specific namespace
./ionos-cloud-watchdog --namespace my-namespace

# JSON output
./ionos-cloud-watchdog --output json
```

### Options

```
-kubeconfig string   path to kubeconfig file
-namespace string    kubernetes namespace to check (default: all)
-output string       output format: text or json (default "text")
-verbose             verbose output
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

## License

MIT
