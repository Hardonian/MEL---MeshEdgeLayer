# MEL Operator TUI: Terminal-Based Control Plane

For headless or low-bandwidth environments, MEL provides a high-fidelity terminal user interface (TUI). It is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a fluid, reactive experience.

## TUI Architecture & Flow

```mermaid
graph TD
    subgraph "MEL CLI (Operator)"
        TUI[Interactive TUI]
        CMD[CLI Commands]
    end

    subgraph "MEL TUI Model"
        H[Header]
        T[Tabs]
        C[Content Area]
        F[Footer / Status]
    end

    TUI --> H
    TUI --> T
    TUI --> C
    TUI --> F
    
    C --> V1[Overview]
    C --> V2[Mesh Topology]
    C --> V3[Active Alerts]
    C --> V4[Control Posture]
```

## TUI Experience Vision

The TUI maintains a "Retro-Futurist" technical aesthetic to ensure clarity and high-contrast legibility in terminal environments.

![MEL TUI Vision](/c:/Users/scott/.gemini/antigravity/brain/3857245b-4abd-4d41-9b8b-41da0a674b43/mel_tui_mockup_retro_future_terminal_1774057747796.png)

## Navigation Shortcuts

| Key | Action |
| :--- | :--- |
| **`Tab`** / **`→`** | Advance to the next tab. |
| **`←`** | Go back to the previous tab. |
| **`1`** - **`6`** | Jump directly to a specific tab. |
| **`R`** | Manually refresh all data segments. |
| **`P`** | Toggle automatic "Live Polling" (default on). |
| **`V`** | (Diagnostics tab only) Trigger a DB Vacuum. |
| **`Q`** / **`Ctrl+C`** | Exit the TUI. |

## Feature Tabs

### 1. OVERVIEW
A high-level health report of all enabled transports and their current ingest status. Displays pending alerts and recent message counts.

### 2. MESH
The "Mesh Inventory". Lists all nodes observed by MEL, their schema versions, and the reported capabilities of each transport (Send, Ingest, Inventory).

### 3. ALERTS
A focused view of all active transport-level failures, including detailed reason codes and the "First Triggered" timestamp.

### 4. CONTROL
Displays the current [Control Mode](file:///c:/Users/scott/GitHub/MEL-MeshEdgeLayer/docs/architecture/control-plane.md) and any active remediation episodes.

### 5. LOGS
Historical incident log for the current session. Shows status changes across segments and nodes.

### 6. DIAGS
Low-level system information, including memory allocation, uptime, platform details, and the physical location of the SQLite database.

*MEL — Truthful Local-First Mesh Observability.*
