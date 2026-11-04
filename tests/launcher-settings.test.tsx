import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { LauncherSettingsView } from "../src/launcher-settings-view";
import { i18n } from "../src/i18n";
import type { SettingsState } from "../src/settings";

function render(files: boolean, weChat = false, isMacOS = true, busy = false) {
  const state: SettingsState = {
    launcher: { enabled: true, files, weChat, shortcut: "Alt+Space", spotlightConflict: false, indexing: false, fileCount: 2, error: "" },
    launcherBusy: busy, launcherError: "", startup: null, startupBusy: false, startupError: "",
    preferences: null, menuBarDraft: [], trayPreview: null, languageDraft: "en", updateProxyDraft: false,
    hideDockDraft: false, preferencesBusy: false, preferencesError: "",
  };
  return renderToStaticMarkup(<LauncherSettingsView state={state} isMacOS={isMacOS} onLauncherShortcut={() => {}} />);
}

test("WeChat search explains the session limit and requires ordinary file search", async () => {
  await i18n.changeLanguage("en");
  const off = render(false);
  expect(off).toContain("this session only");
  expect(off).toContain("next launch will not scan WeChat directories");
  expect(off).toMatch(/<button[^>]*disabled=""[^>]*aria-checked="false"[^>]*data-action="toggle-launcher-wechat"/);
  expect(render(true, true)).toMatch(/<button[^>]*aria-checked="true"[^>]*data-action="toggle-launcher-wechat"/);
  expect(render(true, false, true, true)).toMatch(/<button[^>]*disabled=""[^>]*data-action="toggle-launcher-wechat"/);
  expect(render(true, false, false)).not.toContain("toggle-launcher-wechat");
  expect(render(true, false, false)).not.toContain("iCloud");
});
