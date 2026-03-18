# Install MEL on Linux

## Direct-node-first install

```bash
make build
sudo install -Dm755 ./bin/mel /usr/local/bin/mel
sudo install -d -m 0750 /etc/mel /var/lib/mel
sudo useradd --system --home /var/lib/mel --shell /usr/sbin/nologin mel 2>/dev/null || true
sudo cp configs/mel.serial.example.json /etc/mel/mel.json
sudo chown root:root /etc/mel/mel.json
sudo chmod 600 /etc/mel/mel.json
sudo chown root:root /var/lib/mel
sudo /usr/local/bin/mel config validate --config /etc/mel/mel.json
sudo /usr/local/bin/mel doctor --config /etc/mel/mel.json
```

Then edit `/etc/mel/mel.json`:

- set `storage.data_dir` and `storage.database_path` to your real persistent paths,
- set `serial_device` to the real attached node path,
- or switch to `configs/mel.tcp.example.json` if your deployment uses a Meshtastic-compatible TCP stream.

## Service user and serial permissions

If MEL runs as a dedicated service account, create it first and then grant serial access as appropriate for the distro:

```bash
sudo useradd --system --home /var/lib/mel --shell /usr/sbin/nologin mel || true
sudo usermod -aG dialout mel
```

Use `uucp` instead of `dialout` on distros that expose serial devices through that group.

## Service install

Use `docs/ops/systemd/mel.service` as the baseline unit. Set `User=` / `Group=` to an account that can:

- read `/etc/mel/mel.json`,
- write the MEL data directory,
- open the configured serial device when using direct-node serial mode.
