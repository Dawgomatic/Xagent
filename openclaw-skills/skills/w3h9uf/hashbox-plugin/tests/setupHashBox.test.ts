import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { readFile, mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { configureHashBox } from "../src/setupHashBox.js";

let tempDir: string;

describe("configureHashBox", () => {
  beforeEach(async () => {
    tempDir = await mkdtemp(join(tmpdir(), "hashbox-setup-"));
    vi.spyOn(process, "cwd").mockReturnValue(tempDir);
  });

  afterEach(async () => {
    vi.restoreAllMocks();
    await rm(tempDir, { recursive: true, force: true });
  });

  it("should save a valid token and return success message", async () => {
    const result = await configureHashBox("HB-test-token-123");
    expect(result).toContain("HashBox configured successfully");
  });

  it("should throw on invalid token without HB- prefix", async () => {
    await expect(configureHashBox("INVALID-token")).rejects.toThrow(
      "Invalid token"
    );
  });

  it("should write a readable config file after save", async () => {
    await configureHashBox("HB-readable-token");
    const configPath = join(tempDir, "hashbox_config.json");
    const raw = await readFile(configPath, "utf-8");
    const config = JSON.parse(raw) as { token: string };
    expect(config.token).toBe("HB-readable-token");
  });

  it("should throw on empty string", async () => {
    await expect(configureHashBox("")).rejects.toThrow("Invalid token");
  });
});
