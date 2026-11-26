<p align="center"> <img src="web/static/icons/icon-192-c4f1f45902a0.png" alt="Health Balance icon" width="128"> </p>

# Health Balance

A self-hosted web app that tracks your **Biological Reserve Score**, a weekly estimate of your long-term health reserve, from your health data and gym log. Nothing is typed in.

- **Weekly score**: starts at 1000 and moves 12% toward a target each week, after an aging tax.
- **Markers**: VO2 max, waist-to-height, resting heart rate, HRV, blood pressure, cardio recovery.
- **Behaviours** (4-week average): sleep, steps, exercise, gym working sets, mindfulness.
- **History**: weekly breakdowns and daily trends.
- **AI insights**: a weekly summary written by Claude.

Tap **?** in the app for how the score is calculated.

## Setup

1. Next to [docker-compose.yml](docker-compose.yml), create a `.env`:
   ```bash
   INGEST_TOKEN=a_long_random_string
   ANTHROPIC_API_KEY=your_key_here
   ```
2. Run `docker-compose up -d`.
3. Open `http://<your-server>:8080`, add it to your Home Screen, and fill in your profile in **Settings**.

| Variable | Default | Purpose |
| --- | --- | --- |
| `INGEST_TOKEN` | | Required to receive health data. |
| `ANTHROPIC_API_KEY` | | Enables AI insights. |
| `ANTHROPIC_MODEL` | `claude-opus-5-5` | Model for insights. |
| `DB_PATH` | `./data/health.db` | Database file. |
| `PORT` | `8080` | Listen port. |

## Sending health data

Post your samples once a day. Send the last 7 days each time; days already stored are overwritten.

```
POST /api/ingest/health
Authorization: Bearer <INGEST_TOKEN>
Content-Type: application/json

{"rows": "KEY;value;unit;start;end;source\n…"}
```

One sample per line, up to 5 MB per request. `start` and `end` are ISO 8601 with a UTC offset, and a sample counts toward the date it starts on. Decimal commas are accepted. Sources whose name contains `watch` or `iphone` are grouped as watch and phone.

| KEY | Measure | Units | Daily value |
| --- | --- | --- | --- |
| `RHR` | Resting heart rate | count/min, bpm | latest |
| `VO2` | VO2 max | mL/(kg·min) | latest |
| `CR` | Cardio recovery | count/min, bpm | latest |
| `HRV` | Heart rate variability | ms | mean |
| `WEIGHT` | Weight | kg, g, lb | mean |
| `WAIST` | Waist | cm, mm, m, in | mean |
| `BPSYS` | Systolic blood pressure | mmHg | mean |
| `BPDIA` | Diastolic blood pressure | mmHg | mean |
| `STEPS` | Steps | count | sum* |
| `EXMIN` | Exercise | min, h, s | sum* |
| `SETS` | Gym working sets | count | sum* |
| `SLEEP` | Sleep stage segment | ignored | see below |
| `MIND` | Mindfulness session | ignored | sessions and minutes |

\* Summed per source; the source with the highest total is kept.

- `SLEEP`: the value is the stage (`inbed`, `asleep`, `core`, `deep`, `rem`, `awake`). A night counts toward the day you wake up; naps are ignored.
- `MIND`: the value is ignored.
- Summed and averaged metrics skip the oldest day of each request, which is usually incomplete.
- The response reports how many day values were stored, and up to 10 rejected lines with the reason.

### History

Older days can be posted the same way, in batches that overlap by one day. To load an Apple Health export instead, copy `export.xml` into `./data` and run:

```bash
docker exec health-balance ./backfill-health -export /app/data/export.xml -db /app/data/health.db
```

Options: `-from YYYY-MM-DD` (default `2021-10-01`), `-tz` (time zone for untagged records, default `Europe/Stockholm`), `-dry-run`.

## Hosting

Run `ghcr.io/agusespa/health-balance` (amd64 and arm64; tags `:X.Y.Z` and `:latest`).

Health Balance has no login of its own. Put it behind an access proxy such as Cloudflare Access, and send health data to it over a private network, for example from an n8n container on the same Docker network. The ingest endpoint also requires `INGEST_TOKEN`.

A sender outside that network, such as a phone posting directly, goes through Access with a **service token**: add a policy with the **Service Auth** action that allows it, and send its `CF-Access-Client-Id` and `CF-Access-Client-Secret` headers with every request.

## Data

Everything lives in `./data/health.db` (SQLite, WAL mode). With AI insights on, your profile, scores and daily values are sent to Claude.

Back up while running:

```bash
sqlite3 ./data/health.db ".backup '/path/to/backup.db'"
```

To restore, stop the app, copy the backup over `./data/health.db`, and start it again.
