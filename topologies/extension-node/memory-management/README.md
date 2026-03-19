# Extension node memory-management assets

Reserve this directory for small, bounded device-side state helpers only when verified code requires them, such as:

- short sensor buffers awaiting uplink
- compact local error logs
- explicit retry state for disrupted uplinks

Do not treat this directory as proof of large persistent storage support on constrained devices.
