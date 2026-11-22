# Agent Coder Team Desktop

Wails 3 + Vue 3 + Element Plus desktop client.

## Development

Start the backend first:

```bash
docker compose up -d backend
```

Run the desktop app:

```bash
wails3 dev
```

Build:

```bash
wails3 build
```

`build/` contains Wails packaging source files such as icons, platform tasks and installer configuration. It is not compiled output. Generated folders such as `bin/`, `frontend/dist/`, `frontend/node_modules/` and `frontend/bindings/` are excluded from Git.

See `../../docs/development.md` for full setup and architecture notes.
