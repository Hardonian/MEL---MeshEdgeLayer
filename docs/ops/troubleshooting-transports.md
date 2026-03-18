# Troubleshooting transports

## `serial device not found`

- Confirm the node is attached.
- Prefer stable `/dev/serial/by-id/...` paths over `/dev/ttyUSB0` when possible.
- Re-run `mel doctor` after reconnecting the device.

## `permission denied reading serial device`

- Add the MEL user to `dialout` or `uucp`.
- Restart the shell or service after the group change.
- Re-run `mel doctor` to confirm the device now opens read/write.

## `connected but idle`

This means MEL connected successfully but has not yet observed a real packet. Check:

- The node is active.
- The transport source is the node you expect.
- Another client is not consuming the stream in a way that prevents MEL from reading it.

## `TCP endpoint unreachable`

- Confirm the host and port are correct.
- Confirm the endpoint really serves a Meshtastic-compatible stream, not an HTTP UI or another protocol.
- Check firewall and bind settings on the remote host.

## Device disappears after reboot or replug

- Use `/dev/serial/by-id/...` in production configs.
- MEL retries automatically, but it cannot recover if the configured path never comes back.
- `mel doctor` will distinguish missing devices from permission failures.
