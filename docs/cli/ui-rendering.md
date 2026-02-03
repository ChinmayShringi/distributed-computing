# UI Rendering

## Overview

The `ui` package provides styled terminal output for the CLI including headers, progress indicators, and themed messages.

## Components

### ASCII Header

```go
ui.RenderHeader()
```

Displays themed welcome banner:

```
╔══════════════════════════════════════════╗
║           OmniForge CLI                  ║
║         AI-Assisted DevOps               ║
╚══════════════════════════════════════════╝
```

### Spinner

```go
spinner := ui.NewSpinner("Processing...")
spinner.Start()
// ... do work
spinner.Stop()
```

With elapsed time:

```
⠋ Processing... (2.5s)
```

### Dangerous Mode Indicator

```go
fmt.Println(ui.RenderDangerousModeIndicator())
```

Output:
```
[DANGEROUS MODE ENABLED]
```

Styled with red/bold ANSI codes.

### Progress Bar

```go
progress := ui.NewProgress(total)
progress.Update(current)
progress.Complete()
```

Output:
```
[████████████░░░░░░░░]  60% (60/100)
```

## Color Schemes

| Element | Color |
|---------|-------|
| User messages | Cyan |
| Assistant | Green |
| System | Yellow |
| Error | Red |
| Warning | Yellow |
| Success | Green |
| Dangerous | Red + Bold |

## ANSI Codes

```go
const (
    Reset   = "\033[0m"
    Red     = "\033[31m"
    Green   = "\033[32m"
    Yellow  = "\033[33m"
    Blue    = "\033[34m"
    Cyan    = "\033[36m"
    Bold    = "\033[1m"
)
```

## No-Color Mode

Disable colors with `--no-color`:

```go
if noColor {
    ui.DisableColors()
}
```

## Role Prefixes

```go
ui.RenderMessage("user", "Hello")    // [You]: Hello
ui.RenderMessage("assistant", "Hi")  // [Assistant]: Hi
ui.RenderMessage("system", "Info")   // [System]: Info
```

## Tool Approval Card

```go
ui.RenderApprovalCard(ApprovalPrompt{
    Tool:      "restart_service",
    Args:      args,
    Why:       "Service is not responding",
    RiskLevel: RiskHigh,
})
```

Renders bordered card with tool details and options.

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/ui/render.go` | Main rendering |
| `internal/ui/spinner.go` | Spinner component |
| `internal/ui/progress.go` | Progress bar |
| `internal/ui/colors.go` | Color definitions |
