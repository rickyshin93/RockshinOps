import { copyFileSync, mkdirSync, readdirSync } from "node:fs";
import { join } from "node:path";

const target = process.argv[2];
const suffixes: Record<string, string[]> = {
  "darwin/arm64": [".dmg"],
  "darwin/amd64": [".dmg"],
  "windows/amd64": [".exe"],
  "linux/amd64": [".deb", ".tar.gz"],
};
if (!target || !suffixes[target]) throw new Error(`Unsupported release target: ${target}`);
const directory = join("build", target.replace("/", "-"));
const files = readdirSync(directory).filter(file => suffixes[target]!.some(suffix => file.endsWith(suffix)) && (target !== "windows/amd64" || file.includes(" Setup ")));
for (const suffix of suffixes[target]!) {
  if (!files.some(file => file.endsWith(suffix))) throw new Error(`Missing ${suffix} package in ${directory}`);
}
mkdirSync("release-files", { recursive: true });
const assetNames = files.map(file => file.replaceAll(" ", "."));
for (const [index, file] of files.entries()) {
  copyFileSync(join(directory, file), join("release-files", assetNames[index]!));
}
console.log(`Collected ${target}: ${assetNames.join(", ")}`);
