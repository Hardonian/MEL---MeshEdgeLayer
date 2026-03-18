# Install on Linux

```bash
make build
sudo PREFIX=/usr/local CONFIG_DIR=/etc/mel DATA_DIR=/var/lib/mel ./scripts/install-linux.sh
sudo systemctl daemon-reload
sudo systemctl enable --now mel
```

Then validate:

```bash
sudo /usr/local/bin/mel doctor --config /etc/mel/mel.json
sudo /usr/local/bin/mel status --config /etc/mel/mel.json
```
