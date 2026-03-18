# Privacy audit

Run:

```bash
./bin/mel privacy audit --format text --config /etc/mel/mel.json
./bin/mel privacy audit --format json --config /etc/mel/mel.json
```

The audit emits severity-ranked findings with remediation text. It currently checks:

- remote bind posture
- remote bind without auth
- MQTT encryption requirement
- map reporting
- long retention windows
- precise position storage
- export redaction
- empty trust list
- anonymous MQTT config
- JSON-oriented MQTT topics
