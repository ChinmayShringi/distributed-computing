# Platform Detection

## Overview

The `osdetect` package identifies the operating system, architecture, and distribution for platform-specific operations.

## Detection Results

```go
type SystemInfo struct {
    OS       string // darwin, linux, windows
    Arch     string // amd64, arm64
    Platform string // macos, ubuntu, debian, rhel
    Distro   string // ubuntu, debian (for Linux)
    Version  string // OS version
}
```

## Supported Platforms

| Platform | OS | Detection |
|----------|----|-----------|
| macOS | darwin | `runtime.GOOS` |
| Ubuntu | linux | `/etc/os-release` |
| Debian | linux | `/etc/os-release` |
| RHEL | linux | `/etc/os-release` |
| Windows | windows | `runtime.GOOS` |

## Detection Methods

### macOS

```go
if runtime.GOOS == "darwin" {
    info.Platform = "macos"
    // Get version from sw_vers
}
```

### Linux

Parses `/etc/os-release`:

```bash
ID=ubuntu
VERSION_ID="22.04"
```

```go
file, _ := os.Open("/etc/os-release")
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "ID=") {
        info.Distro = strings.Trim(line[3:], `"`)
    }
}
```

### Architecture

```go
info.Arch = runtime.GOARCH // amd64, arm64
```

## Usage

```go
info, err := osdetect.Detect()
if err != nil {
    return err
}

fmt.Printf("Platform: %s/%s\n", info.OS, info.Arch)

if info.IsSupported() {
    // Proceed with installation
} else {
    return errors.New(info.SupportedMessage())
}
```

## Support Check

```go
func (s *SystemInfo) IsSupported() bool {
    switch s.OS {
    case "darwin":
        return true
    case "linux":
        return s.Distro == "ubuntu" || s.Distro == "debian"
    default:
        return false
    }
}
```

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/osdetect/detect.go` | Main detection |
| `internal/osdetect/linux.go` | Linux-specific |
| `internal/osdetect/darwin.go` | macOS-specific |
