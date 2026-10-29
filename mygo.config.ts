import { defineConfig } from "mygo-cli";
import pkg from "./package.json";

export default defineConfig({
  name: "RockshinOps",
  identifier: "dev.rockshinops.desktop",
  version: pkg.version,
  updates: {
    github: "rickyshin93/RockshinOps",
    publicKey: "kOxLJUSMhx/BmI9BovPOHCPvC0nOV7JXgQ03IsLA/Ik=",
    deltas: 0,
  },
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  resources: ["LICENSE", "THIRD_PARTY_NOTICES.md"],
  bindings: "src/mygo.ts",
  out: "build",
  macos: { infoPlist: { CFBundleName: "RockshinOps", CFBundleDisplayName: "RockshinOps", CFBundleDevelopmentRegion: "en", CFBundleLocalizations: ["en", "zh-Hans"] } },
});
