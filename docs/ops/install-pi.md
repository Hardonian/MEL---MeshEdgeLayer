# Install MEL on Raspberry Pi

1. Build or copy the `mel` binary onto the Pi.
2. Create a dedicated `mel` service account or choose an existing service user.
3. Copy `configs/mel.serial.example.json` to `/etc/mel/mel.json`.
4. Change the data directory to something persistent like `/var/lib/mel`.
5. Set `serial_device` to the real path for the attached node.
6. Add the chosen MEL service user to `dialout`.
7. Run `mel doctor --config /etc/mel/mel.json`.
8. Start `mel serve --config /etc/mel/mel.json` or install the systemd service.

If the USB path changes after reboot, prefer `/dev/serial/by-id/...` in the config.
