import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { writeFile, mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { sendHashBoxNotification } from "../src/pushToHashBox.js";
import type { MetricEntry, AuditEntry } from "../src/types.js";

let tempDir: string;

const EXPECTED_TOKEN = "HB-valid-token";
const EXPECTED_URL = `https://webhook-vcphors6kq-uc.a.run.app/webhook?token=${EXPECTED_TOKEN}`;

describe("sendHashBoxNotification", () => {
  beforeEach(async () => {
    tempDir = await mkdtemp(join(tmpdir(), "hashbox-push-"));
    vi.spyOn(process, "cwd").mockReturnValue(tempDir);
  });

  afterEach(async () => {
    vi.restoreAllMocks();
    await rm(tempDir, { recursive: true, force: true });
  });

  it("should POST article payload to correct URL with token as query param", async () => {
    const configPath = join(tempDir, "hashbox_config.json");
    await writeFile(configPath, JSON.stringify({ token: EXPECTED_TOKEN }), "utf-8");

    const mockResponse = { ok: true, status: 200 } as Response;
    const fetchMock = vi.fn().mockResolvedValue(mockResponse);
    vi.stubGlobal("fetch", fetchMock);

    const result = await sendHashBoxNotification(
      "article", "News", "📰", "Test Article", "This is a test body"
    );

    expect(result.status).toBe(200);
    expect(result.message).toBe("Notification sent successfully");

    expect(fetchMock).toHaveBeenCalledOnce();
    const [calledUrl, calledOptions] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(calledUrl).toBe(EXPECTED_URL);
    expect(calledOptions.method).toBe("POST");

    const sentBody = JSON.parse(calledOptions.body as string);
    expect(sentBody).toEqual({
      type: "article",
      channelName: "News",
      channelIcon: "📰",
      title: "Test Article",
      body: "This is a test body",
    });
  });

  it("should POST metric payload with channelName and channelIcon", async () => {
    const configPath = join(tempDir, "hashbox_config.json");
    await writeFile(configPath, JSON.stringify({ token: EXPECTED_TOKEN }), "utf-8");

    const mockResponse = { ok: true, status: 200 } as Response;
    const fetchMock = vi.fn().mockResolvedValue(mockResponse);
    vi.stubGlobal("fetch", fetchMock);

    const metrics: MetricEntry[] = [
      { label: "CPU", value: 85, unit: "%" },
    ];

    const result = await sendHashBoxNotification(
      "metric", "System Monitor", "📊", "Server Stats", metrics
    );

    expect(result.status).toBe(200);

    const sentBody = JSON.parse(
      (fetchMock.mock.calls[0] as [string, RequestInit])[1].body as string
    );
    expect(sentBody).toEqual({
      type: "metric",
      channelName: "System Monitor",
      channelIcon: "📊",
      title: "Server Stats",
      metrics,
    });
  });

  it("should POST audit payload with channelName and channelIcon", async () => {
    const configPath = join(tempDir, "hashbox_config.json");
    await writeFile(configPath, JSON.stringify({ token: EXPECTED_TOKEN }), "utf-8");

    const mockResponse = { ok: true, status: 200 } as Response;
    const fetchMock = vi.fn().mockResolvedValue(mockResponse);
    vi.stubGlobal("fetch", fetchMock);

    const entries: AuditEntry[] = [
      { timestamp: "2026-02-19T00:00:00Z", event: "email_changed", severity: "info", details: "a@b.com -> c@d.com" },
    ];

    const result = await sendHashBoxNotification(
      "audit", "Audit Log", "🔍", "User Change", entries
    );

    expect(result.status).toBe(200);

    const sentBody = JSON.parse(
      (fetchMock.mock.calls[0] as [string, RequestInit])[1].body as string
    );
    expect(sentBody).toEqual({
      type: "audit",
      channelName: "Audit Log",
      channelIcon: "🔍",
      title: "User Change",
      entries,
    });
  });

  it("should throw when config file is missing", async () => {
    await expect(
      sendHashBoxNotification("article", "Ch", "📰", "Title", "Body")
    ).rejects.toThrow("HashBox config not found");
  });

  it("should handle network failure gracefully", async () => {
    const configPath = join(tempDir, "hashbox_config.json");
    await writeFile(configPath, JSON.stringify({ token: EXPECTED_TOKEN }), "utf-8");

    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("fetch failed")));

    const result = await sendHashBoxNotification(
      "article", "News", "📰", "Test", "Body"
    );
    expect(result.status).toBe(0);
    expect(result.message).toContain("Network error");
    expect(result.message).toContain("fetch failed");
  });

  it("should handle non-ok response status", async () => {
    const configPath = join(tempDir, "hashbox_config.json");
    await writeFile(configPath, JSON.stringify({ token: EXPECTED_TOKEN }), "utf-8");

    const mockResponse = { ok: false, status: 403 } as Response;
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(mockResponse));

    const result = await sendHashBoxNotification(
      "article", "News", "📰", "Test", "Body"
    );
    expect(result.status).toBe(403);
    expect(result.message).toBe("Request failed with status 403");
  });

  it("should not include token in the request body", async () => {
    const configPath = join(tempDir, "hashbox_config.json");
    await writeFile(configPath, JSON.stringify({ token: EXPECTED_TOKEN }), "utf-8");

    const mockResponse = { ok: true, status: 200 } as Response;
    const fetchMock = vi.fn().mockResolvedValue(mockResponse);
    vi.stubGlobal("fetch", fetchMock);

    await sendHashBoxNotification(
      "article", "News", "📰", "Test", "Body"
    );

    const sentBody = JSON.parse(
      (fetchMock.mock.calls[0] as [string, RequestInit])[1].body as string
    );
    expect(sentBody).not.toHaveProperty("token");
  });
});
