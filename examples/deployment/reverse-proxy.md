# Reverse proxy in front of MEL

MEL’s HTTP server is a single listener (`bind.api`). For remote access:

1. Enable `auth.enabled` and set strong `session_secret`, UI credentials, and/or `MEL_AUTH_API_KEYS`.
2. Do **not** rely on `auth.allow_insecure_remote`; terminate TLS at nginx, Caddy, or another proxy and forward to `127.0.0.1:8080`.
3. Keep `bind.allow_remote` false unless you have a deliberate reason to expose the Go listener directly.

Example nginx location (illustrative):

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Preserve `X-API-Key` from clients if you use API key auth.
