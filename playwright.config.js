const path = require("path");
const { defineConfig, devices } = require("@playwright/test");

const e2eTmpDir = path.join(__dirname, "e2e", ".tmp");
const dbPath = path.join(e2eTmpDir, "health-balance-e2e.db");
const port = process.env.PLAYWRIGHT_PORT || "4300";
const baseURL = `http://127.0.0.1:${port}`;

module.exports = defineConfig({
  testDir: path.join(__dirname, "e2e", "tests"),
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL,
    trace: "on-first-retry",
  },
  webServer: {
    command: `go run ./cmd/server`,
    url: `${baseURL}/health`,
    reuseExistingServer: !process.env.CI,
    env: {
      ...process.env,
      DB_PATH: dbPath,
      GOCACHE: "/tmp/health-balance-go-cache",
      HOST: "127.0.0.1",
      PORT: port,
    },
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        serviceWorkers: "block",
      },
    },
  ],
});
