# Artifact Download

## Overview

The `artifact` command downloads release artifacts from the OmniForge server with SHA256 verification.

## Usage

```bash
# Download latest release
omniforge artifact download --latest

# Auto-approve prompts
omniforge artifact download --latest -y

# Execute install script after download
omniforge artifact download --latest --install
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--latest` | | Download latest release |
| `--install` | | Run install script after extraction |
| `--yes` | `-y` | Auto-approve prompts |

## Download Process

1. **Get manifest** - Fetch latest release metadata
2. **Download artifact** - Stream to local file with progress
3. **Verify SHA256** - Compare hash with manifest
4. **Extract** - Unpack to version directory
5. **Install** (optional) - Run platform install script

## Output

```
Fetching latest release manifest...
  Version: v1.2.3
  Artifact: hostagent-server.tar.gz
  Size: 150.5 MB
  SHA256: abc123...

Downloading...
  [████████████████████] 100% (150.5 MB)

Verifying SHA256...
  ✓ Hash matches

Extracting to ~/.spotlight/releases/v1.2.3/...
  ✓ Extraction complete

Run install script? [y/N]: y
Running install script...
  ✓ Installation complete
```

## Manifest Structure

```json
{
  "version": "v1.2.3",
  "artifacts": [
    {
      "id": "artifact_abc123",
      "name": "hostagent-server.tar.gz",
      "platform": "darwin",
      "arch": "arm64",
      "size": 157810688,
      "sha256": "abc123def456...",
      "download_url": "https://..."
    }
  ],
  "install_script": "install.sh"
}
```

## Storage Locations

| Item | Location |
|------|----------|
| Downloaded artifacts | `~/.omniforge/cache/` |
| Extracted releases | `~/.spotlight/releases/{version}/` |
| Install scripts | Inside extracted directory |

## SHA256 Verification

```go
func verifyHash(filePath string, expectedHash string) error {
    file, _ := os.Open(filePath)
    hasher := sha256.New()
    io.Copy(hasher, file)
    actualHash := hex.EncodeToString(hasher.Sum(nil))

    if actualHash != expectedHash {
        return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
    }
    return nil
}
```

## Progress Display

```
Downloading...
  [████████░░░░░░░░░░░░]  42% (63.2 MB / 150.5 MB)
```

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| "not authenticated" | No session | Run `omniforge login` |
| "artifact not found" | No release for platform | Check platform support |
| "hash mismatch" | Corrupted download | Retry download |
| "extraction failed" | Disk full or permissions | Check disk space |

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | artifact command |
| `internal/artifact/download.go` | Download with progress |
| `internal/artifact/verify.go` | SHA256 verification |
| `internal/artifact/extract.go` | Tarball extraction |
| `internal/api/artifacts.go` | Manifest fetching |
