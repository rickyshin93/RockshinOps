import { defineConfig } from "mygo-cli";
import pkg from "./package.json";

export default defineConfig({
  name: "LocalOps",
  identifier: "dev.localops.desktop",
  version: pkg.version,
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  resources: ["LICENSE", "THIRD_PARTY_NOTICES.md"],
  bindings: "src/mygo.ts",
  out: "build",
  macos: { infoPlist: { CFBundleDevelopmentRegion: "en", CFBundleLocalizations: ["en", "zh-Hans"] } },
});
