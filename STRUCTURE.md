# Project structure

<!--
Filled in once the structure has been agreed on, then kept up to date by whoever changes the code
(see "Keep the agent docs in sync" in AGENTS.md). It describes what the project is NOW, not plans.
Delete the sections that don't apply.
-->

<PROJECT> <one or two sentences: what it does, for whom, how the pieces talk to each other>.

```text
repo/
├── <dir>/            <language/framework + version> — <responsibility>
├── Dockerfile        <stages>
├── docker-compose.yml
├── .env.example      every supported env var, documented; `.env` is git-ignored
└── Makefile          dev commands (run `make help`)
```

## <Component> (`<dir>/src`)

```text
<dir>/src/
├── main.<ext>        wiring: env, config, state, server
├── config/           config read once from env
├── errors/           error types → API responses
└── …
```

### API

All under `<BASE>/api`:

- `GET /health` → `{status}`
- `<METHOD> /<path>` → `<response>`: <notes: auth, status codes, limits>

### Data and state

<!-- Databases, caches, files on disk: where they live, what is durable and what is cache, key formats, TTLs. -->

### Configuration

| Env var | Default | Meaning |
| --- | --- | --- |
| `PORT` | `8080` | |

## Frontend (`frontend/src`)

| File | Responsibility |
| --- | --- |
| `pages/…` | |
| `lib/api.ts` | API types (mirror the backend models), fetch helpers |
| `lib/i18n.ts` | EN/IT dictionaries |
