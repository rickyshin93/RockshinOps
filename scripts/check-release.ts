import pkg from "../package.json";
const tag = process.argv[2];
if (tag !== `v${pkg.version}`) throw new Error(`Release tag ${tag} must match package.json version v${pkg.version}`);
