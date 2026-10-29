import { createHash, createPublicKey, verify } from "node:crypto";
import { copyFileSync, mkdirSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import config from "../mygo.config";

const suffixes: Record<string, string[]> = {
  "darwin/arm64": [".dmg"],
  "darwin/amd64": [".dmg"],
  "windows/amd64": [".exe"],
  "linux/amd64": [".deb", ".tar.gz"],
};

type UpdateManifest = { version: string; url: string; size: number; signature: string };

export function verifyUpdate(manifest: UpdateManifest, archive: Uint8Array, publicKey: string, version: string, url: string) {
  if (manifest.version !== version || manifest.url !== url) throw new Error("Update version or download URL does not match this release");
  if (manifest.size !== archive.byteLength) throw new Error("Update archive size does not match manifest");
  const key = createPublicKey({ format: "der", type: "spki", key: Buffer.concat([
    Buffer.from("302a300506032b6570032100", "hex"), Buffer.from(publicKey, "base64"),
  ]) });
  if (!verify(null, createHash("sha256").update(archive).digest(), key, Buffer.from(manifest.signature, "base64"))) {
    throw new Error("Update archive signature is invalid");
  }
}

export function collectRelease(target: string, signed: boolean, directory = join("build", target.replace("/", "-")), destination = "release-files") {
  if (!suffixes[target]) throw new Error(`Unsupported release target: ${target}`);
  const files = readdirSync(directory).filter(file => suffixes[target]!.some(suffix => file.endsWith(suffix)) && (target !== "windows/amd64" || file.includes(" Setup ")));
  for (const suffix of suffixes[target]!) {
    if (!files.some(file => file.endsWith(suffix))) throw new Error(`Missing ${suffix} package in ${directory}`);
  }
  if (signed) {
    const platform = target.replace("/", "-");
    const manifestName = `update-${platform}.json`;
    const archiveName = `rockshinops-${config.version}-${platform}.tar.gz`;
    const manifest: UpdateManifest = JSON.parse(readFileSync(join(directory, manifestName), "utf8"));
    verifyUpdate(manifest, readFileSync(join(directory, archiveName)), config.updates!.publicKey, config.version!,
      `https://github.com/${config.updates!.github}/releases/download/v${config.version}/${archiveName}`);
    files.push(manifestName, archiveName);
  }
  mkdirSync(destination, { recursive: true });
  for (const file of new Set(files)) copyFileSync(join(directory, file), join(destination, file.replaceAll(" ", ".")));
  console.log(`Collected ${target}${signed ? " with verified signed update" : ""}`);
}

if (import.meta.main) {
  const target = process.argv[2];
  if (!target) throw new Error("Usage: collect-release.ts <target> [--signed]");
  collectRelease(target, process.argv.includes("--signed"));
}
