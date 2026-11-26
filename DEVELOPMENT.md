# Development

```bash
make seed      # profile and ~2.5 years of synthetic data in ./data/health.db
make dev       # Go API on :8080 and the SvelteKit dev server on :5173
make test      # Go tests
make e2e       # Playwright tests against a seeded production build
make backfill EXPORT=~/Downloads/apple_health_export/export.xml   # load an Apple Health export locally
```

The frontend lives in `web/` (SvelteKit, static adapter). `make web` builds it into `internal/ui/dist`, which the Go binary embeds.

The ingest endpoint, `POST /api/ingest/health`, takes `{"rows": "KEY;value;unit;start;end;source\n…"}` with `Authorization: Bearer $INGEST_TOKEN`. Parsing and unit conversion are in `internal/ingest`.
