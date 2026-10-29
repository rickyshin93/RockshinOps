import { readFileSync } from "node:fs";
import pkg from "../package.json";

export function checkRelease(tag: string | undefined, version: string, changelog: string) {
  if (tag !== `v${version}`) throw new Error(`Release tag ${tag} must match package.json version v${version}`);
  const section = changelog.split(/^## /m).find(section => section.startsWith(`${version}\n`));
  if (!section?.slice(version.length).trim()) throw new Error(`CHANGELOG.md has no notes for ${version}`);
}

if (import.meta.main) checkRelease(process.argv[2], pkg.version, readFileSync(new URL("../CHANGELOG.md", import.meta.url), "utf8"));
