# Linux Install

```bash
make build
mkdir -p /var/lib/mel
cp configs/mel.example.json /etc/mel.json
./bin/mel serve --config /etc/mel.json
```
