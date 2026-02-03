# macOS Setup Guide

This guide covers the specific requirements for running OmniForge and hostagent-server on macOS.

## Prerequisites

### 1. Homebrew (Required)

Homebrew is required for installing dependencies on macOS. Most Mac users don't have it pre-installed.

**Check if Homebrew is installed:**
```bash
which brew
```

**Install Homebrew if not present:**
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

After installation, follow the instructions to add Homebrew to your PATH:
- Apple Silicon (M1/M2/M3): Add to `~/.zprofile`: `eval "$(/opt/homebrew/bin/brew shellenv)"`
- Intel Mac: Add to `~/.zprofile`: `eval "$(/usr/local/bin/brew shellenv)"`

### 2. Docker Desktop (Required)

Install Docker Desktop for Mac from https://www.docker.com/products/docker-desktop/

Or via Homebrew:
```bash
brew install --cask docker
```

## Critical Docker Desktop Configuration

**IMPORTANT:** Before hostagent-server can work on macOS, you MUST configure these Docker Desktop settings manually. These cannot be automated.

### Step 1: Enable /opt in File Sharing

The hostagent-server installs to `/opt/hostagent-server`. Docker needs permission to access this path.

1. Open **Docker Desktop**
2. Click the **gear icon** (Settings)
3. Go to **Resources** → **File Sharing**
4. Click the **+** button
5. Add `/opt` to the list
6. Click **Apply & Restart**

### Step 2: Enable Host Networking

Host networking is required for the hostagent-server containers to communicate properly.

1. Open **Docker Desktop**
2. Click the **gear icon** (Settings)
3. Go to **Network**
4. Check **Enable Host Networking**
5. Click **Apply & Restart**

### Step 3: Wait for Docker to Restart

After applying settings, wait for Docker Desktop to fully restart before proceeding with installation.

Verify Docker is running:
```bash
docker info
```

## Installing hostagent-server

Once Docker Desktop is configured:

```bash
# Login to OmniForge
omniforge login

# Install hostagent-server
omniforge install
```

Or use the chat assistant:
```bash
omniforge chat
> setup hostagent server
```

## Troubleshooting

### "Cannot create directory /opt/hostagent-server"

Docker doesn't have permission to write to `/opt`. Add `/opt` to Docker Desktop's File Sharing settings.

### Containers can't communicate / port binding issues

Enable Host Networking in Docker Desktop settings.

### "brew: command not found"

Install Homebrew first:
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### Docker commands fail after installation

Restart your terminal or run:
```bash
# Apple Silicon
eval "$(/opt/homebrew/bin/brew shellenv)"

# Intel
eval "$(/usr/local/bin/brew shellenv)"
```

## Differences from Linux

| Aspect | Linux | macOS |
|--------|-------|-------|
| Package Manager | apt/yum/dnf | Homebrew |
| Docker | Native | Docker Desktop (VM) |
| File Sharing | Native | Requires configuration |
| Host Networking | Native | Requires enabling |
| Install Path | /opt (direct) | /opt (via Docker VM) |

## Verification

After setup, verify everything is working:

```bash
# Check Docker
docker info

# Check hostagent-server containers
docker ps --filter name=hostagent

# Check health
omniforge doctor
```
