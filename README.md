# Longevity Health Tracker

A comprehensive longevity tracking system built with Go, SQLite, and HTMX. Track your health across three pillars and watch your master score evolve over time.

## Features

- **Master Score Algorithm**: Starting at 1000 points, your score evolves weekly based on performance and aging.
- **Three Pillar System**:
  - 🏥 **Health Pillar**: Sleep, WHtR, RHR, Nutrition.
  - 💪 **Fitness Pillar**: VO2 Max, Workouts, Steps, Mobility, Recovery.
  - 🧠 **Cognition Pillar**: Memory, Reaction Time, Mindfulness.
- **Aging Rate**: Automatic weekly decay based on your age.
- **SQLite Database**: All data persisted for backups and analysis.
- **Real-time Updates**: HTMX-powered interface with no page reloads.

## Quick Start (Docker)

You can run the application directly using Docker. Create a `docker-compose.yml` file with the following content:

```yaml
services:
  app:
    image: ghcr.io/agusespa/health-balance:latest
    ports:
      - "8080:8080"
    environment:
      - DB_PATH=/data/health.db
    volumes:
      - ./data:/data
```

Then run:

```bash
docker-compose up -d
```

Access the application at [http://localhost:8080](http://localhost:8080).

## Data Persistence

The application uses a **Bind Mount** to ensure your health data persists on your host machine.

- **Host Path**: `./data` (relative to your `docker-compose.yml`)
- **Mount Point**: `/data` (inside the container)
- **Database File**: `./data/health.db`

## Production-Ready Backups

The database is configured with **Write-Ahead Logging (WAL)** mode, which allows for safe "hot" backups while the application is running.

### Atomic Backups to NAS

To perform a safe, "hot" backup from your host (e.g., to a NAS), use the `sqlite3` command directly on your host machine. This ensures the backup is consistent and avoids corruption.

```bash
# Atomic, high-performance backup directly from the host
sqlite3 ./data/health.db ".backup '/path/to/your/nas/health_backup.db'"
```

> [!TIP]
> You can automate this by adding the command above to your host's crontab:
> ```bash
> 0 3 * * * sqlite3 /path/to/health-balance/data/health.db ".backup '/path/to/nas/health_backup_$(date +\%Y\%m\%d).db'"
> ```

## Key Algorithms

### Pillar Scoring
- **Health**: Focuses on Sleep, WHtR (Waist-to-Height Ratio), RHR, and Nutrition.
- **Fitness**: Tracks VO2 Max, Workouts, Steps, Mobility, and Cardio Recovery.
- **Cognition**: Measures Working Memory (Dual N-Back), Reaction Time, and Mindfulness.

### Master Score
Your score starts at 1000 and is updated weekly. Every Sunday, the "Aging Tax" (based on your age) is deducted, and your weekly performance points from the three pillars are added to calculate your new Master Score.
