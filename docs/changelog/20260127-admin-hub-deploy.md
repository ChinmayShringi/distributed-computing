# Admin Hub Migration

**Date**: 2026-01-27

## Summary

Replaced the old static admin UI (`server/admin/index.html`) with a new React-based admin panel (`admin-hub/`). The new admin panel is built with Vite, React, TypeScript, and shadcn-ui.

## Changes

### New Features

- **Incidents Page**: View and inspect system incidents with details
- **Allowlist Page**: View tool allowlist with status and risk indicators
- **Artifacts Page**: View artifact manifest and download binaries

### Architecture Improvements

- **Clean service architecture**: All API calls moved to dedicated service files
- **HTTP client wrapper**: Centralized HTTP handling with automatic token refresh
- **Runtime configuration**: API base URL derived at runtime, not hardcoded
- **Modular components**: Reusable shared components (PageHeader, ErrorBanner, LoadingState, EmptyState)

### Technical Changes

1. **New folder structure under `admin-hub/src/`**:
   - `config/` - Runtime configuration
   - `constants/` - API endpoints, storage keys
   - `types/` - TypeScript interfaces
   - `services/` - API service layer
   - `components/shared/` - Reusable components
   - `components/{feature}/` - Feature-specific components

2. **Removed**: `admin-hub/src/lib/api.ts` (replaced by services/)

3. **Go server**: Updated to support SPA routing (fallback to index.html for client-side routes)

4. **GitHub Actions**: Updated to build admin-hub and deploy to server/admin/

## Deployment

### Prerequisites

- Node.js 20+
- npm

### Local Development

```bash
# Start backend
cd server && go run ./cmd/server/main.go

# Start frontend (with proxy to backend)
cd admin-hub && npm run dev
# Visit http://localhost:8080/admin/
```

### Production Build

```bash
cd admin-hub
npm ci
npm run build

# Copy to server
rm -rf ../server/admin/*
cp -r dist/* ../server/admin/

# Restart server
```

### CI/CD

The GitHub Actions workflow automatically:
1. Builds admin-hub with `npm ci && npm run build`
2. Packages dist as tarball
3. Copies to EC2 and extracts to `server/admin/`
4. Restarts the omniforge service

## Environment Variables

No new environment variables required for the frontend. All configuration is derived at runtime.

Backend env vars (unchanged):
- `PORT` - Server port (default: 3000)
- `JWT_SECRET` - JWT signing secret
- `DB_*` - Database connection

## Verification Checklist

- [ ] `GET /admin/` loads React app
- [ ] Direct refresh on `/admin/users` works (SPA routing)
- [ ] Login with real credentials works
- [ ] Dashboard shows server status
- [ ] Incidents page loads data
- [ ] Allowlist page loads data
- [ ] Artifacts page shows manifest
- [ ] Knowledge upload works
- [ ] Chat sends messages
- [ ] Logout clears session

## Breaking Changes

- The old `server/admin/index.html` has been replaced
- API client is now in `services/` instead of `lib/api.ts`
- Token storage keys changed from `auth_token` to `access_token` and `refresh_token`

## Files Changed

### New Files (admin-hub)

| Path | Purpose |
|------|---------|
| `src/config/index.ts` | Runtime config |
| `src/constants/index.ts` | API endpoints, constants |
| `src/types/index.ts` | TypeScript interfaces |
| `src/services/httpClient.ts` | HTTP wrapper |
| `src/services/authService.ts` | Auth API |
| `src/services/incidentsService.ts` | Incidents API |
| `src/services/allowlistService.ts` | Allowlist API |
| `src/services/artifactsService.ts` | Artifacts API |
| `src/services/usersService.ts` | Users API |
| `src/services/knowledgeService.ts` | Knowledge API |
| `src/services/mcpService.ts` | MCP API |
| `src/services/serverService.ts` | Server API |
| `src/services/chatService.ts` | Chat API |
| `src/services/index.ts` | Service exports |
| `src/components/shared/*.tsx` | Shared components |
| `src/components/incidents/*.tsx` | Incident components |
| `src/components/allowlist/*.tsx` | Allowlist components |
| `src/components/artifacts/*.tsx` | Artifact components |
| `src/pages/Incidents.tsx` | Incidents page |
| `src/pages/Allowlist.tsx` | Allowlist page |
| `src/pages/Artifacts.tsx` | Artifacts page |

### Modified Files

| Path | Changes |
|------|---------|
| `admin-hub/vite.config.ts` | Added `base: '/admin/'`, dev proxy |
| `admin-hub/src/App.tsx` | Added new routes |
| `admin-hub/src/components/layout/AppSidebar.tsx` | Added nav items |
| `admin-hub/src/contexts/AuthContext.tsx` | Use authService |
| `admin-hub/src/pages/*.tsx` | Updated to use services |
| `server/cmd/server/main.go` | SPA fallback routing |
| `.github/workflows/deploy.yml` | Build admin-hub step |

### Deleted Files

| Path | Reason |
|------|--------|
| `admin-hub/src/lib/api.ts` | Replaced by services |
