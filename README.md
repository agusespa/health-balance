# Health Balance Tracker

It keeps track of a Biological Reserve Score, a weighted algorithm that measures functional longevity. It moves beyond simple activity tracking to assess Intrinsic Capacity —the total sum of your physical and mental resources.

### Features
- **Overall Score Algorithm**: Starting at 1000 points, your score evolves weekly based on performance and aging.
- **Three Pillar System**:
  - **Health Pillar**: Sleep, WHtR, RHR, Nutrition.
  - **Fitness Pillar**: VO2 Max, Workouts, Steps, Mobility, Recovery.
  - **Cognition Pillar**: Memory, Reaction Time, Mindfulness.
- **Aging Rate**: Automatic weekly decay based on your age.

## Quick Start

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

## Backups

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

---
# Biological Reserve Score (BRS): Executive Summary

## Health Pillar
**Measuring the efficiency of internal repair and metabolic state.**
- Sleep Variance: Uses a baseline of 75. The formula calculates the delta between your weekly average and this threshold. Positive deltas represent a "repair surplus," while negative deltas represent a "repair deficit."
- Structural Ratio (WHtR): Acts as a high-weight static variable. By multiplying the difference between the user’s ratio and the 0.48 baseline by 1000, small improvements in body composition result in significant, long-term score gains.
- Resting Heart Rate (RHR): Functions as a relative comparison. It measures the deviation from a 3-month rolling average, allowing the algorithm to detect systemic strain relative to the user's personal norm.

## Fitness Pillar
**Measuring cardiovascular ceiling and structural resilience.**
- VO2 Max: This is the most heavily weighted function (x20). It uses an Age-Adjusted Baseline to ensure the score remains relevant to the user's current physiological potential.
- Cardio Recovery: Measures the delta of heart rate drop in 60 seconds against a 20 BPM baseline. This function captures the speed of the nervous system's transition from an active to a restful state.
- Frequency Metrics (Workouts/Steps/Mobility): These function as "operational inputs." They use simple count-based baselines (e.g., 3 workouts, 8,000 steps) to provide the consistent point-flow needed to offset the weekly entropy tax.

## Cognition Pillar
**Measuring neural efficiency and mental capacity.**
- Working Memory (Dual N-Back): Calculated as a deviation from Level 2. As the user's "N-Level" increases, the cognitive contribution to the master score grows linearly, representing an expansion of mental processing reserve.
- Reaction Speed: Measures the difference between a 3-month baseline and current performance in milliseconds. This captures acute changes in neural processing speed, often used to identify weeks of high neurological fatigue.
- Mindfulness Frequency: A frequency-based function that rewards consistency. It uses a baseline of 3 sessions to represent the minimum input required for cognitive regulation.

## The Master Algorithm: Entropy & Buffers
The system is built on the interaction between two opposing forces: Decay and Performance.
- The Entropy Function (The Aging Tax): 
$$Tax = \frac{(Age / 200)}{52}$$
This function creates a weekly score reduction that scales with age. It establishes a "moving baseline" where older users require higher performance inputs to maintain score stability compared to younger users.
- The Stability Buffer: High-value structural metrics ($VO_2$ Max and WHtR) function as filters. Because they are slow to change, they provide a mathematical "floor" that prevents the score from fluctuating wildly based on a single week's behavior.
