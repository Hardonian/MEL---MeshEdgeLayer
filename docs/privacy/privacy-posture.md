# Privacy posture

MEL defaults to:

- localhost bind
- redacted exports
- no precise position storage
- no map reporting
- MQTT encryption required in policy posture
- short retention windows

Operators can override these defaults, but MEL surfaces the consequences through `mel doctor`, `mel config validate`, `mel privacy audit`, `/api/v1/privacy/audit`, and the local UI.
