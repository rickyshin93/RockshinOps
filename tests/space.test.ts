import { expect, test } from "bun:test";
import { toggleAllAppFiles, reconcileAppFiles } from "../src/app-selection";
import { renderBody, type ViewState } from "../src/views";
import { installedAppsView } from "../src/app-uninstall";

const uninstall = { apps: [], review: null, selection: [], busy: false, searching: false, message: "", outcome: null };
const state = { page: "apps", info: { scanTargets: [] }, snapshot: {} } as unknown as ViewState;

test("shows app candidates while the full search continues and prevents premature removal", () => {
  const html = installedAppsView("", { ...uninstall, busy: true, searching: true,
    message: "", selection: ["/Applications/Example.app"], review: {
      app: { name: "Example", path: "/Applications/Example.app" }, bundleId: "example.app", reviewId: 1,
      files: [{ path: "/Applications/Example.app", reason: "应用包" }], scanIssues: [],
    },
  });
  expect(html).toContain("/Applications/Example.app");
  expect(html).toContain("Searching for more associated files");
  expect(html).toContain('data-action="remove-app-files" aria-haspopup="dialog" disabled');
  expect(html).toContain('data-action="stop-app-search"');
  expect(html).toContain('data-app-select-all="true" aria-label="Select all associated items" checked');
});

test("reports inaccessible scan locations separately from app candidates and keeps reviewed removal available", () => {
  const html = installedAppsView("", { ...uninstall,
    busy: false, searching: false, message: "", selection: ["/Applications/Example.app"], review: {
      app: { name: "Example", path: "/Applications/Example.app" }, bundleId: "example.app", reviewId: 1,
      files: [{ path: "/Applications/Example.app", reason: "应用包" }],
      scanIssues: [{ path: "/Library/Protected <folder>", kind: "permission", error: "operation not permitted" }],
    },
  });
  expect(html).toContain('<details class="uninstall-coverage">');
  expect(html).toContain("Scan coverage is limited");
  expect(html).toContain("Unread locations: 1");
  expect(html).toContain("/Library/Protected &lt;folder&gt;");
  expect(html).toContain("operation not permitted");
  expect(html).toContain("Full Disk Access");
  expect(html).toContain('data-action="open-full-disk-access"');
  expect(html.indexOf('data-action="open-full-disk-access"')).toBeLessThan(html.indexOf('<details class="uninstall-coverage">'));
  expect(html).toContain("quit and reopen RockshinOps");
  expect(html).not.toContain('data-action="remove-app-files" aria-haspopup="dialog" disabled');
  expect(html).not.toContain("items could not be read; search results are incomplete");
});

test("offers Full Disk Access settings before choosing an installed app on macOS", () => {
  const html = installedAppsView("", { ...uninstall,
    busy: false, review: null, message: "", apps: [],
  });
  expect(html).toContain('data-action="open-full-disk-access"');
});

test("does not suggest a permission grant for a location that changed during the search", () => {
  const html = installedAppsView("", { ...uninstall,
    busy: false, searching: false, message: "", selection: [], review: {
      app: { name: "Example", path: "/Applications/Example.app" }, bundleId: "example.app", reviewId: 1,
      files: [{ path: "/Applications/Example.app", reason: "应用包" }],
      scanIssues: [{ path: "/Library/Changed", kind: "changed", error: "no such file or directory" }],
    },
  });
  expect(html).not.toContain('data-action="open-full-disk-access"');
});

test("keeps macOS permission settings out of Windows and Linux app views", () => {
  for (const platform of ["windows", "linux"]) {
    const html = renderBody({ ...state, page: "apps", info: { machine: { platform } }, appMode: true,
      processError: "", processes: { items: [] }, appBusy: false, appReview: null,
      appMessage: "", installedApps: [], query: "",
    } as unknown as ViewState);
    expect(html).not.toContain('data-action="open-full-disk-access"');
  }
});


test("select all includes later discoveries, and deselect all clears them", () => {
  const first = ["/app", "/cache"];
  const selected = toggleAllAppFiles(first, false);
  expect(selected).toEqual(first);
  const expanded = reconcileAppFiles([...first, "/later"], selected, true);
  expect(expanded).toEqual([...first, "/later"]);
  expect(toggleAllAppFiles(expanded, true)).toEqual([]);
  expect(reconcileAppFiles(["/app"], selected, false)).toEqual(["/app"]);
});
