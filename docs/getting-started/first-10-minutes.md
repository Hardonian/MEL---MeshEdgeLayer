# First 10 Minutes with MEL

Welcome, operator. This guide gets you from zero to a running MEL instance in under 10 minutes.

---

## 🏃 Minute 0-2: Verify Installation

### Check the binary

```bash
mel version
```

**Expected output:**
```text
0.1.0-rc1
```

### Verify help

```bash
mel
```

This lists all available subcommands. If this works, MEL is correctly installed on your PATH.

---

## 📝 Minute 2-4: Bootstrapping Config

### Create initial config

```bash
mel init --config ./mel.json
```

**Expected output:**
```json
{
  "status": "initialized",
  "config": "./mel.json",
  "bind": ":8080",
  "database": "data/mel.db"
}
```

### Review and Hardening

```bash
cat mel.json
```

Key fields:
- `storage.database_path`: Where packets live.
- `bind.api`: Where the API and dashboard are served.

**Hardening (Recommended):**
```bash
chmod 600 mel.json
```

---

## 🩺 Minute 4-6: The Doctor Path

MEL is unique in that it includes a "Doctor" that verifies everything—from SQLite write permissions to transport reachability—before you even start the server.

### Run system checks

```bash
mel doctor --config ./mel.json
```

**What the doctor checks:**
- **Config syntax**: Validates your `mel.json`.
- **Database**: Verifies SQLite read/write/re-read cycles.
- **Privacy**: Scans for potentially unsafe mesh configurations.
- **Transports**: Probes reachability for configured nodes/brokers.

If the doctor gives you a `PASS` for the Core system, you are ready.

---

## ⚡ Minute 6-8: Launch Ingest

### Start the server

```bash
mel serve --config ./mel.json
```

**What happens now:**
1. MEL boots its transport workers.
2. It attempts to connect to any enabled transports.
3. The Internal API server starts.
4. If `features.web_ui` is true, the dashboard goes live.

---

## 🕶️ Minute 8-10: Verify Observability

### Use the CLI Status

```bash
mel status --config ./mel.json
```

This gives you a real-time snapshot of uptime, packet counts, and transport health.

### View the Instrument Panel

```bash
mel panel --config ./mel.json
```

This is the TUI (Terminal UI) view for field operators. It summarizes current mesh health at a glance.

### Access the Web Dashboard

Open your browser to: **http://localhost:8080**

You should see the MEL landing page. If you have an active transport (like a Serial node connected), you will see live packets appearing in the ledger.

---

## 💡 Next Steps

- **Configure Transports**: Edit `mel.json` to add Serial, TCP, or MQTT paths.
- **Run a Privacy Audit**: `mel privacy audit --config ./mel.json`.
- **Export Data**: `mel export --config ./mel.json --out bundle.json`.

---

*MEL — Truthful local-first mesh observability.*
