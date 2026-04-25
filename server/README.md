# Sally Server

## Run

```bash
cd /home/wyatt/sally/server
go run ./cmd/sally-server
```

The server listens on `:8080`.

## Health Check

```bash
curl -i http://localhost:8080/healthz
```

Expected:

- `HTTP/1.1 200 OK`
- body: `ok`

For LAN access from another machine, use the host IP instead, for example:

```bash
curl -i http://10.0.0.104:8080/healthz
```

## OpenAI Config

The real provider is selected only when both of these are set:

- `OPENAI_API_KEY`
- `OPENAI_MODEL`

If neither is set, the server uses the stub extractor.

If only one is set, the server exits at startup.
