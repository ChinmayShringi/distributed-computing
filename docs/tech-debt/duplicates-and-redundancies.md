# Duplicates and Redundancies Analysis

This document catalogs duplicate implementations, redundant code patterns, and opportunities for consolidation in the OmniForge CLI codebase.

## Priority Legend

| Priority | Description |
|----------|-------------|
| **HIGH** | Causes bugs, maintenance burden, or confusion. Address immediately. |
| **MEDIUM** | Creates technical debt but not immediately harmful. Plan to fix. |
| **LOW** | Minor inconsistency. Fix opportunistically. |

---

## 1. Duplicate Artifact Data Structures

**Priority:** HIGH

**Files:**
- `internal/artifact/download.go:156-171`
- `internal/api/artifacts.go:13-29`
- `server/internal/artifacts/github.go:26-43`

**Issue:**
Three separate `Manifest` and `Artifact` struct definitions:

```go
// internal/artifact/download.go - For tar.gz manifest
type Manifest struct {
    Version     string
    Name        string
    Prereq      map[string]Script
    Install     map[string]Script
}

// internal/api/artifacts.go - For API responses
type Manifest struct {
    Version   string
    Artifacts []Artifact
}

// server/internal/artifacts/github.go - For GitHub releases
type Manifest struct {
    Version   string
    Artifacts []Artifact
}
```

**Recommendation:**
- Create `internal/types/artifact.go` with shared definitions
- Use type aliases or composition for context-specific extensions

---

## 2. Duplicate Download Mechanisms

**Priority:** HIGH

**Files:**
- `internal/artifact/download.go` - Download with SHA256 verification + progress
- `internal/api/artifacts.go` - DownloadArtifact (simple io.Copy)
- `server/internal/artifacts/github.go` - DownloadAsset (GitHub API)
- `internal/omni/install.go` - tar extraction via exec

**Issue:**
Multiple ways to download/extract artifacts with inconsistent:
- Error handling
- Progress reporting
- Hash verification

**Recommendation:**
- Use `artifact.Downloader.Download()` as the primary mechanism
- Refactor `api.Client.DownloadArtifact()` to use `Downloader`
- Keep `artifacts.DownloadAsset()` only for GitHub-specific operations

---

## 3. Duplicate Chat Action Execution Loops

**Priority:** HIGH

**Files:**
- `cmd/spotlight/commands/root.go:1227-1422` (`runChatLoop()`)
- `cmd/spotlight/commands/root.go:1535-1676` (`runSingleChat()`)

**Issue:**
~400 lines of nearly identical code for:
- Loading approval rules
- Processing actions with interactive approval
- Handling DecisionYes, DecisionAlwaysAllow, DecisionNo, DecisionCancel

**Recommendation:**
Extract to shared function:
```go
func processActions(ctx context.Context, actions []Action, rules *approval.Rules, registry *tools.Registry) error
```

---

## 4. Duplicate Default Path Constants

**Priority:** MEDIUM

**Files:**
- `internal/config/config.go:20`
- `internal/omni/omni.go:15`

**Issue:**
```go
// config.go
DefaultOMNIInstallPath = "/opt/hostagent-server"

// omni.go
DefaultInstallPath = "/opt/hostagent-server"
```

Same value, different variable names in different files.

**Recommendation:**
Create `internal/constants/paths.go`:
```go
const DefaultInstallPath = "/opt/hostagent-server"
```

---

## 5. Duplicate Dangerous Mode Flag Handling

**Priority:** MEDIUM

**Files:**
- `cmd/spotlight/commands/root.go:576-595` (doctorCmd)
- `cmd/spotlight/commands/root.go:1061-1080` (chatCmd)

**Issue:**
Identical flag parsing and consent prompting code:
```go
allowDangerous, _ := cmd.Flags().GetBool("allow-dangerous")
adAlias, _ := cmd.Flags().GetBool("ad")
if allowDangerous || adAlias {
    if !autoConfirm {
        if !mode.PromptDangerousConsent() {
            return fmt.Errorf("...")
        }
    }
    modeCtx, err = mode.NewDangerousContext(Version)
}
```

**Recommendation:**
Extract to helper:
```go
func initializeDangerousMode(cmd *cobra.Command, version string) (*mode.ModeContext, error)
```

---

## 6. ~~Duplicate Health/Status Check Systems~~ ✅ RESOLVED

**Priority:** ~~MEDIUM~~ **RESOLVED**

**Files:**
- `internal/omni/health.go` - CheckHealth (now uses real docker inspect)
- `internal/omni/lifecycle.go` - GetStatus (service status)
- `internal/status/status.go` - Comprehensive status report

**Resolution:**
- Fixed `getContainerHealth()` stub to use real `docker inspect` command
- `health.go` now properly checks container health status
- Added `WaitForHealthy()` function with retry logic for post-install checks
- `omniforge install` now waits for services to become healthy before reporting

---

## 7. Duplicate Consent/Approval UI Mechanisms

**Priority:** MEDIUM

**Files:**
- `internal/mode/consent.go` - PromptDangerousConsent (full-screen)
- `internal/approval/approval.go` - PromptApproval (interactive)
- `internal/artifact/install.go` - promptYesNo (simple)

**Issue:**
Three different user prompting patterns with different:
- UI styles
- Input handling
- Error behavior

**Recommendation:**
Create `internal/ui/prompt.go` with unified prompt functions:
```go
func PromptYesNo(question string) bool
func PromptWithWarning(title, warning string) bool
func PromptInteractive(options []Option) Decision
```

---

## 8. ~~Duplicate Installation Lifecycle Patterns~~ ✅ RESOLVED

**Priority:** ~~MEDIUM~~ **RESOLVED**

**Files:**
- `internal/omni/` (Controller pattern)
- `internal/artifact/install.go` (Artifact-based installation)
- `internal/tools/registry.go` (SetupHostAgentServerTool)

**Resolution:**
- Unified to single artifact-based installation approach
- Deleted script-based installers (`internal/installers/` package)
- Both CLI `omniforge install` and LLM agent use `setup_hostagent_server` tool
- Tool downloads artifact via AWS S3 presigned URL, verifies SHA256, runs manifest scripts

---

## 9. Duplicate Configuration Systems

**Priority:** MEDIUM

**Files:**
- `internal/config/config.go` (CLI - JSON file)
- `server/internal/config/config.go` (Server - env vars)

**Issue:**
Both packages define:
- `Config` struct
- `Load()` function
- Default values

Different storage backends but similar patterns.

**Recommendation:**
- Create shared interface `ConfigLoader`
- Keep implementations separate but use consistent patterns

---

## 10. Incomplete/Dead Code

**Priority:** LOW-MEDIUM

**Files:**
- `cmd/spotlight/commands/root.go` - `artifactCmd` (lines 849-1007)
- `cmd/spotlight/commands/root.go` - `bundleCmd` (lines 650-680)

**Issue:**
- `artifact download` command overlaps with `install` command
- `bundle create` has "not yet implemented" message

**Recommendation:**
- Evaluate if `artifact` command is needed or should be removed
- Complete or remove `bundle create` implementation

---

## 11. Duplicate Auth Token Types

**Priority:** LOW

**Files:**
- `internal/api/auth.go` - TokenResponse (client-side)
- `server/internal/auth/jwt.go` - TokenPair (server-side)

**Issue:**
Similar token structures defined in both client and server code:
```go
// Client
type TokenResponse struct {
    AccessToken, RefreshToken, ExpiresIn, TokenType
}

// Server
type TokenPair struct {
    AccessToken, RefreshToken, ExpiresIn, TokenType
}
```

**Recommendation:**
Create shared types in `internal/types/auth.go` or a shared module.

---

## 12. Duplicate Incident Collection Patterns

**Priority:** LOW

**Files:**
- `internal/incident/collector.go` - CollectX functions
- `internal/bundle/incident.go` - Incident bundling

**Issue:**
Two packages handle incident data:
- `incident/` - Collects diagnostic data
- `bundle/` - Creates and uploads bundles

Similar error handling and data gathering patterns.

**Recommendation:**
Unify interfaces but keep logical separation.

---

## Summary

| # | Issue | Priority | Files | Status |
|---|-------|----------|-------|--------|
| 1 | Artifact data structures | HIGH | 3 | Open |
| 2 | Download mechanisms | HIGH | 4 | Open |
| 3 | Chat action loops | HIGH | 1 | Open |
| 4 | Path constants | MEDIUM | 2 | Open |
| 5 | Dangerous mode flags | MEDIUM | 1 | Open |
| 6 | Health/status checks | ~~MEDIUM~~ | 3 | ✅ **RESOLVED** |
| 7 | Consent/approval UI | MEDIUM | 3 | Open |
| 8 | Installation patterns | ~~MEDIUM~~ | 2 | ✅ **RESOLVED** |
| 9 | Configuration systems | MEDIUM | 2 | Open |
| 10 | Dead/incomplete code | LOW-MEDIUM | 1 | Open |
| 11 | Auth token types | LOW | 2 | Open |
| 12 | Incident collection | LOW | 2 | Open |

---

## Action Plan

### Phase 1: Quick Wins
- [ ] Create shared constants file for paths
- [ ] Extract dangerous mode initialization helper
- [ ] Remove or complete dead code (artifact, bundle commands)

### Phase 2: High Priority
- [ ] Consolidate artifact data structures to shared package
- [ ] Extract chat action processing to shared function
- [ ] Unify download mechanisms around `artifact.Downloader`

### Phase 3: Medium Priority
- [x] ~~Create shared health check utilities~~ ✅ DONE
- [ ] Standardize consent/approval UI patterns
- [x] ~~Unify installation interfaces~~ ✅ DONE

### Phase 4: Low Priority (Ongoing)
- [ ] Share auth token types
- [ ] Standardize incident collection patterns
- [ ] Unify configuration interfaces

---

## Completed Work

### 1. Unified Installation Tool ✅
- Renamed `setup_hostagent_server_recipe` → `setup_hostagent_server`
- Deleted script-based tools (`setup_hostagent_server_via_script`, `run_hostagent_server_via_script`)
- Both CLI `omniforge install` and LLM agent now use the same artifact-based approach with AWS S3 presigned URLs
- Deleted `internal/installers/` package and `scripts/installers/hostagent-server/` directory

### 2. Auth Token Bug Fixes ✅
- Server: `/v1/auth/refresh` now returns `refresh_token` in response
- CLI: `IsAuthenticated()` validates both access and refresh tokens exist

### 3. Unified Health Check System ✅
- Fixed `getContainerHealth()` stub to use real `docker inspect` command
- Health checks now properly report container health status (not "unknown")
- Added `WaitForHealthy()` function with retry logic (5s interval, configurable timeout)
- `omniforge install` now waits up to 90 seconds for services to become healthy
- PostgreSQL health check handles "none" status (containers without healthcheck defined)

### 4. Deleted Obsolete Code ✅
- `internal/installers/` package (entire directory)
- `scripts/installers/hostagent-server/` directory
- `scripts/installers/` directory (empty after cleanup)
- Server `/scripts/installers/` endpoint and handler
- Obsolete documentation files

### 5. CI/CD Pipeline Updated ✅
- Removed installer scripts smoke test
- Removed scripts directory copy step
- Updated planner.go AI prompt to use unified tools