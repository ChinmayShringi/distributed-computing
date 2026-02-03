# Admin Hub CI/CD Setup

**Date**: 2026-01-27

## Summary

Configured CI/CD for the admin-hub frontend to automatically build, test, and deploy to the same path as the old admin UI (`/admin/`).

## CI/CD Pipeline

### Trigger

- **Deploy**: Automatic on push to `main` branch
- **CI Tests**: On all pushes and pull requests to `main`

### Workflow Files

| File | Purpose |
|------|---------|
| `.github/workflows/ci.yml` | Build, lint, and test admin-hub + Go binaries |
| `.github/workflows/deploy.yml` | Build and deploy to EC2 |

### CI Pipeline (`ci.yml`)

```
┌─────────────────┐
│ test-admin-hub  │ ─── npm ci, lint, build
├─────────────────┤
│ test-cli        │ ─── go test ./...
├─────────────────┤
│ test-server     │ ─── go test ./... (server module)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     build       │ ─── Build all binaries (waits for all tests)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   smoke-test    │ ─── Start server, check health
└─────────────────┘
```

### Deploy Pipeline (`deploy.yml`)

```
┌─────────────────┐
│   checkout      │
├─────────────────┤
│  setup node.js  │
├─────────────────┤
│ build admin-hub │ ─── npm ci && npm run build
├─────────────────┤
│  package dist   │ ─── tar -czf admin-hub-dist.tar.gz
├─────────────────┤
│  deploy to EC2  │ ─── SCP + SSH + systemctl
├─────────────────┤
│  smoke tests    │ ─── Health, admin panel, SPA routing
└─────────────────┘
```

## Secrets Required

| Secret | Description |
|--------|-------------|
| `EC2_SSH_KEY` | SSH private key for EC2 access |
| `EC2_HOST` | EC2 public IP or hostname |
| `GH_PAT` | GitHub PAT for cloning repo on EC2 |

## Hosting

| Component | Location |
|-----------|----------|
| **Server** | EC2 instance |
| **Admin Files** | `/opt/omniforge/admin/` |
| **Service** | `omniforge` systemd unit |
| **Admin URL** | `http://<EC2_HOST>:3000/admin/` |

## SPA Routing

The Go server handles SPA routing with fallback to `index.html`:

```go
// server/cmd/server/main.go
mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/admin/")
    if path == "" {
        path = "index.html"
    }
    fullPath := filepath.Join(adminDir, path)
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        // SPA fallback
        http.ServeFile(w, r, filepath.Join(adminDir, "index.html"))
        return
    }
    http.ServeFile(w, r, fullPath)
})
```

This ensures:
- `/admin/` → `index.html`
- `/admin/incidents` → `index.html` (React Router handles it)
- `/admin/assets/index.js` → actual file

## Smoke Tests

After each deploy, these checks run:

1. **Health check**: `GET /health` returns 200
2. **Admin panel**: `GET /admin/` returns 200 with `<div id="root">`
3. **SPA routing**: `GET /admin/incidents` returns 200 (not 404)

## How to Deploy

### Automatic (Recommended)

Push to `main` branch:
```bash
git push origin main
```

The deploy workflow will:
1. Build admin-hub
2. Deploy to EC2
3. Run smoke tests

### Manual

```bash
# Build locally
cd admin-hub
npm ci
npm run build

# Copy to EC2
tar -czf admin-hub-dist.tar.gz -C dist .
scp admin-hub-dist.tar.gz ubuntu@$EC2_HOST:/tmp/

# SSH and extract
ssh ubuntu@$EC2_HOST "
  cd /opt/omniforge/repo
  rm -rf server/admin/*
  mkdir -p server/admin
  tar -xzf /tmp/admin-hub-dist.tar.gz -C server/admin
  rm /tmp/admin-hub-dist.tar.gz
  sudo systemctl restart omniforge
"
```

## Rollback Plan

### Quick Rollback (< 5 minutes)

If the new admin-hub fails, revert to the previous working commit:

```bash
# Find last working commit
git log --oneline -10

# Revert to that commit
git revert HEAD
git push origin main
```

The deploy workflow will redeploy the previous version.

### Manual Rollback

SSH to EC2 and restore from git history:

```bash
ssh ubuntu@$EC2_HOST "
  cd /opt/omniforge/repo

  # Find last working commit with admin-hub
  git log --oneline -10 -- admin-hub/

  # Checkout that version of admin-hub
  git checkout <COMMIT_SHA> -- admin-hub/

  # Rebuild and deploy
  cd admin-hub
  npm ci
  npm run build
  rm -rf ../server/admin/*
  cp -r dist/* ../server/admin/

  # Restart
  sudo systemctl restart omniforge
"
```

### Restore Old Static Admin (Emergency)

If admin-hub is completely broken and you need the old static admin:

```bash
ssh ubuntu@$EC2_HOST "
  cd /opt/omniforge/repo

  # Checkout old admin from before migration
  git log --oneline --all -- server/admin/index.html
  git checkout <OLD_COMMIT> -- server/admin/

  sudo systemctl restart omniforge
"
```

Note: The old admin had hardcoded demo credentials and no real backend integration.

## Monitoring

Check deploy status:
- GitHub Actions: https://github.com/<org>/<repo>/actions

Check service status on EC2:
```bash
ssh ubuntu@$EC2_HOST "sudo systemctl status omniforge"
```

Check logs:
```bash
ssh ubuntu@$EC2_HOST "sudo journalctl -u omniforge -f"
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| Deploy fails at npm ci | Corrupted node_modules | Delete `admin-hub/node_modules`, push, retry |
| 404 on /admin/ | Dist not copied | Check deploy logs, verify files in server/admin/ |
| 404 on /admin/route | SPA fallback broken | Check Go server code for admin handler |
| Smoke test fails | Service not started | Check systemctl logs, verify port 3000 |
| Old admin shows | Cache | Clear browser cache, verify dist timestamp |
