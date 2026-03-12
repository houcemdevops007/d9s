# d9s - Docker TUI (Terminal User Interface)

**Author**: KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD

**d9s** is a powerful, lightning-fast, and intuitive Terminal User Interface (TUI) for end-to-end management of your Docker ecosystem. Designed specifically for DevOps, SecOps engineers, and Developers, `d9s` combines container lifecycle management, Docker Compose orchestration, and seamless integration with advanced security scanners (Trivy and Snyk) into a single keyboard-driven interface.

## 🚀 Key Features

### 1. Multi-Host Remote Management (TCP/Unix)
`d9s` is not limited to your local Docker daemon (Unix socket). It dynamically manages multiple standalone Docker endpoints using **Contexts**:
- **Dynamic Configuration**: By tweaking `~/.config/d9s/config.json`, you can save remote hosts (e.g., `tcp://172.16.50.13:2375`).
- **Instant Switching**: Through the left panel "CONTEXTS", users can navigate and actively switch from one Docker host to another (local, remote servers, staging, production) simply by pressing the **Enter** key, with zero downtime or need to restart the application.

### 2. Centralized Docker Resource Management
Manage your Docker environment flawlessly, without leaving the terminal:
- **Containers (key `c`)**: List all actively running and inactive containers. Quick shortcut bindings to Stop (`x`), Restart (`r`), or Remove/Delete (`R` or `Del`).
- **Interactive Shell (key `S`)**: Drop right into a container's command-line shell session instantaneously.
- **Images (key `g` or `i`)**: Explore and manage local images.
- **Volumes (key `v`)** & **Networks (key `n`)**: Comprehensive views for volume and network management.
- **Inspect (key `i`)**: Fetch clean JSON outputs for any selected resource inside the inspect panel.
- **Live Logs (key `l`)**: Tail streams of container logs.
- **Events & Stats**: Real-time monitoring metrics overlaying CPU and memory utilization.

### 3. Native Docker Compose (Projects) Integration
The “PROJECTS” panel bridges native container controls with Docker Compose projects:
- Scans and detects Compose environments (from the host).
- Full orchestration shortcut bindings:
  - `u` = Compose Up
  - `d` = Compose Down
  - `p` = Compose Pull
  - `b` = Compose Build

### 4. Built-in SecOps & Compliance (Trivy + Snyk)
The true strength of `d9s` lies in its native security tabs. Select a Docker Image or Container and:
- **Trivy Scan**: Triggers an open-source scanner generating immediate vulnerability reports broken down by severely (Critical, High, Medium, Low) providing CVE identification.
- **Snyk Scan**: Uses the industry standard Snyk tool (if locally installed and authenticated) for in-depth image analysis and remediation reporting.
- **Best Practices Engine**: An internal heuristic evaluation mechanism that checks for common design flaws (e.g., processes running as `root`, outdated non-LTS versions, huge image sizes, unprotected exposed ports) using cross-correlated inspect data and scanner results!

## 💻 Keyboard Shortcuts
The total TUI experience is optimized for keyboard power-users. No mouse is needed.
Rely on the `Tab` key to pivot between focus panels and sub-tabs, `Arrow Keys` to traverse records, and the `/` key to kick off on-the-fly fuzzy searches against any loaded list.

## 🛠️ Multi-Platform Compilation (Linux & MacOS)

The repository provides a `Makefile` intended to abstract complex cross-compilation target architectures seamlessly.

**Prerequisites**: Go `1.22+` installed.

**Building from source:**
```bash
# Build for your current local OS architecture
make build

# Build specifically for Linux (Outputs AMD64 & ARM64)
make build-linux

# Build specifically for macOS (Outputs AMD64 & ARM64)
make build-darwin

# Target everything (All Architectures & OS binaries generated)
make build-all
```
All compiled binaries will be safely placed inside the `/build` directory.

**To install instantly:**
```bash
sudo make install
# This moves the binary payload straight to /usr/local/bin/d9s
```

> **Note**: For scanning tabs to operate natively, the host OS executing `d9s` must have `trivy` and `snyk` CLIs downloaded, installed into their respective `$PATH`, and authenticated.
