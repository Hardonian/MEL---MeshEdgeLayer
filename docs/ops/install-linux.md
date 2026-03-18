# Install MEL on Linux

## Direct-node-first install

```bash
make build
sudo install -Dm755 ./bin/mel /usr/local/bin/mel
sudo install -d -m 0750 /etc/mel /var/lib/mel
sudo useradd --system --home /var/lib/mel --shell /usr/sbin/nologin mel 2>/dev/null || true
sudo cp configs/mel.serial.example.json /etc/mel/mel.json
sudo chown root:mel /etc/mel/mel.json
sudo chmod 0640 /etc/mel/mel.json
sudo usermod -aG mel,dialout mel
sudo chown -R mel:mel /var/lib/mel
sudo /usr/local/bin/mel doctor --config /etc/mel/mel.json
```

Update `/etc/mel/mel.json` to point at the real serial device or switch to `configs/mel.tcp.example.json` if your deployment uses a Meshtastic-compatible TCP endpoint.

## Service install

Use `docs/ops/systemd/mel.service` as the baseline unit. The shipped unit assumes `User=mel` and `Group=mel`, so create that account or change the unit consistently with your own service user.
