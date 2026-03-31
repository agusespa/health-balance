const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const repoRoot = path.resolve(__dirname, "..", "..");
const tmpDir = path.join(repoRoot, "e2e", ".tmp");
const dbPath = path.join(tmpDir, "health-balance-e2e.db");

function ensureTmpDir() {
  fs.mkdirSync(tmpDir, { recursive: true });
}

function resetSeededDatabase() {
  ensureTmpDir();
  execFileSync(
    "go",
    ["run", "./cmd/seed/main.go", "-db", dbPath, "-reset"],
    {
      cwd: repoRoot,
      env: process.env,
      stdio: "pipe",
    }
  );
}

module.exports = {
  dbPath,
  ensureTmpDir,
  resetSeededDatabase,
};
