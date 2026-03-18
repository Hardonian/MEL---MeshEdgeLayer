# Install on Raspberry Pi (arm64)

Pi installs follow the same path as generic Linux. Build or copy the arm64 binary, then use the Linux installer script.

Recommended checks:

```bash
uname -m
file ./bin/mel
./bin/mel doctor --config /etc/mel/mel.json
```
