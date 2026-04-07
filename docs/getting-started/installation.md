# Installation Guide

MEL is designed for reliability and ease of deployment on various Linux-based platforms. This guide covers the prerequisites and installation steps for supported environments.

## Prerequisites

Regardless of your platform, ensure the following are available:

- **Operating System**: Linux (x86_64, arm64, or armv7).
- **SQLite**: `sqlite3` binary in your `$PATH` (version 3.35.0 or newer).
- **Go Toolchain**: Go 1.24+ (only required if building from source).
- **Permissions**: Root access is not strictly required, but a dedicated service user is recommended for production.

---

## 1. Install MEL

### Option A: From Source (Recommended for Dev/Testing)

```bash
git clone https://github.com/Hardonian/MEL-MeshEdgeLayer.git
cd mel
make build
sudo install -m 755 bin/mel /usr/local/bin/mel
```

### Option B: From Binary

Download the latest release for your architecture from the [Releases](https://github.com/Hardonian/MEL-MeshEdgeLayer/releases) page.

Canonical URL list: [docs/repo-os/canonical-github.md](../repo-os/canonical-github.md).

```bash
tar -xzf mel-linux-amd64-v*.tar.gz
sudo install -m 755 mel /usr/local/bin/mel
```

---

## 2. Environment Setup

For a production-grade deployment, create a dedicated service user and data directories:

```bash
# Create service user
sudo useradd --system --home /var/lib/mel --shell /usr/sbin/nologin mel || true

# Create directories
sudo mkdir -p /etc/mel /var/lib/mel
sudo chown mel:mel /var/lib/mel
sudo chmod 750 /var/lib/mel
```

---

## 3. Platform-Specific Guides

### Raspberry Pi

Raspberry Pi is a first-class target for MEL.

1. Follow the **Linux** steps above.
2. Use `/dev/serial/by-id/...` for serial devices to ensure persistence across reboots.
3. Add the `mel` user to the `dialout` group: `sudo usermod -aG dialout mel`.

### Termux (Android)

Termux support is intended for foreground/manual operation only.

1. Install dependencies: `pkg install golang sqlite`
2. Build from source.
3. Run using `./scripts/termux-run.sh`.

*Note: MEL does not guarantee background persistence on Android.*

---

## 4. Initial Configuration

Generate a default config and restrict its permissions immediately:

```bash
mel init --config /etc/mel/mel.json
sudo chown root:root /etc/mel/mel.json
sudo chmod 600 /etc/mel/mel.json
```

**CRITICAL**: MEL will refuse to start if the configuration file is group- or world-readable in production mode.

---

## 5. Verification

Before starting the service, run the "Doctor" to verify your environment is correctly configured:

```bash
mel doctor --config /etc/mel/mel.json
```

If everything looks good, you are ready to [Start MEL](first-10-minutes.md).
