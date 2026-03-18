# Install MEL on Raspberry Pi

1. Build or copy the `mel` binary onto the Pi.
2. Copy `configs/mel.serial.example.json` to `/etc/mel/mel.json`.
3. Change the data directory to something persistent like `/var/lib/mel`.
4. Set `serial_device` to the real path for the attached node.
5. Add the MEL user to `dialout`.
6. Run `mel doctor --config /etc/mel/mel.json`.
7. Start `mel serve --config /etc/mel/mel.json` or install the systemd service.

If the USB path changes after reboot, prefer `/dev/serial/by-id/...` in the config.
