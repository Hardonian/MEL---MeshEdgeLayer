# MEL FAQ

## General

### What does MEL do?
MEL provides an operator console for mesh networks with explicit degraded state tracking — it doesn't pretend to know things it can't prove.

### How is MEL different from MeshMonitor?
MEL is evidence-first with control operations. MeshMonitor is read-only monitoring.

### What mesh networks?
Primarily Meshtastic. Also supports custom protocols via MQTT.

### Do I needmesh to use MEL?
MEL works with or without meshes. It's also an evidence-first operator console.

## Technical

### What's the tech stack?
- Go binary (operator CLI)
- SQLite (local persistence)  
- Optional PostgreSQL (fleet)
- Web UI (embedded)

### Minimum hardware?
- Raspberry Pi 4 (4GB RAM)
- 16GB SD card
- Serial adapter (optional)

## Support

### How to get help?
- GitHub Issues
- Discord community
- Enterprise: Dedicated CSM
