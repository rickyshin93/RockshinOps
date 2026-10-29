import { test, expect } from "bun:test";
import { createHash, generateKeyPairSync, sign } from "node:crypto";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { collectRelease, verifyUpdate } from "../scripts/collect-release";
import config from "../mygo.config";
import { checkRelease } from "../scripts/check-release";

test("release preflight rejects absent or empty update notes", () => {
  expect(() => checkRelease("v0.2.4", "0.2.4", "# Changelog\n\n## 0.2.3\n\nOld notes")).toThrow("no notes");
  expect(() => checkRelease("v0.2.4", "0.2.4", "# Changelog\n\n## 0.2.4\n\n## 0.2.3\nOld notes")).toThrow("no notes");
  expect(() => checkRelease("v0.2.4", "0.2.4", "# Changelog\n\n## 0.2.4\n\nNew notes")).not.toThrow();
  expect(() => checkRelease("v0.2.4", "0.2.4", "# Changelog\r\n\r\n## 0.2.4\r\n\r\nNew notes")).not.toThrow();
  expect(() => checkRelease("v0.2.3", "0.2.4", "## 0.2.4\nNew notes")).toThrow("must match");
});

test("app and package use the RockshinOps identity", () => {
  expect(config.name).toBe("RockshinOps");
  expect(config.identifier).toBe("dev.rockshinops.desktop");
  expect(config.macos?.infoPlist?.CFBundleDisplayName).toBe("RockshinOps");
});

const { publicKey, privateKey } = generateKeyPairSync("ed25519");
const key = publicKey.export({ format: "der", type: "spki" }).subarray(-32).toString("base64");
const archive = Buffer.from("signed release bytes");
const url = "https://github.com/example/app/releases/download/v0.2.1/app.tar.gz";
const manifest = {
  version: "0.2.1", url, size: archive.length,
  signature: sign(null, createHash("sha256").update(archive).digest(), privateKey).toString("base64"),
};

test("accepts an intact archive signed by the configured key", () => {
  expect(() => verifyUpdate(manifest, archive, key, "0.2.1", url)).not.toThrow();
});

test("rejects corrupted bytes, another signing key, size and release mismatches", () => {
  const corrupted = Buffer.from(archive);
  corrupted[0] ^= 1;
  expect(() => verifyUpdate(manifest, corrupted, key, "0.2.1", url)).toThrow("signature");
  const otherKey = generateKeyPairSync("ed25519").publicKey.export({ format: "der", type: "spki" }).subarray(-32).toString("base64");
  expect(() => verifyUpdate(manifest, archive, otherKey, "0.2.1", url)).toThrow("signature");
  expect(() => verifyUpdate({ ...manifest, size: 1 }, archive, key, "0.2.1", url)).toThrow("size");
  expect(() => verifyUpdate(manifest, archive, key, "0.2.2", url)).toThrow("version");
  expect(() => verifyUpdate(manifest, archive, key, "0.2.1", "https://example.com/app.tar.gz")).toThrow("URL");
});

test("a signed release cannot silently omit a platform manifest", () => {
  mkdirSync("work", { recursive: true });
  const directory = mkdtempSync(join("work", "release-test-"));
  writeFileSync(join(directory, "RockshinOps 0.2.1 arm64.dmg"), "installer");
  expect(() => collectRelease("darwin/arm64", true, directory, join(directory, "out"))).toThrow("update-darwin-arm64.json");
});
