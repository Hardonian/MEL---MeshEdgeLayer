# Install MEL on Linux

## Direct-node-first install

```bash
make build
sudo install -Dm755 ./bin/mel /usr/local/bin/mel
sudo install -d /etc/mel /var/lib/mel
sudo cp configs/mel.serial.example.json /etc/mel/mel.json
sudo chown root:root /etc/mel/mel.json
sudo chmod 600 /etc/mel/mel.json
sudo usermod -aG dialout mel || true
sudo /usr/local/bin/mel doctor --config /etc/mel/mel.json
```

Update `/etc/mel/mel.json` to point at the real serial device or switch to `configs/mel.tcp.example.json` if your deployment uses a Meshtastic-compatible TCP endpoint.

## Service install

Use `docs/ops/systemd/mel.service` as the baseline unit and set `User=` / `Group=` so the service can read the serial device and write the data directory.
