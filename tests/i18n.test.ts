import { afterEach, describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { i18n, resolveLanguage, t } from "../src/i18n";
import { preferencesDirty, type SettingsState } from "../src/settings";
import en from "../internal/localization/en.json";
import zh from "../internal/localization/zh.json";

afterEach(() => { void i18n.changeLanguage("en"); });

describe("language selection", () => {
  test.each([
    [["zh-Hans-CN", "en-US"], "zh"], [["zh-TW"], "zh"], [["en-GB", "zh"], "en"],
    [["fr-FR", "zh-HK"], "zh"], [["fr-FR"], "en"], [[], "en"],
  ] as const)("resolves preferred languages %j", (languages, expected) => {
    expect(resolveLanguage(languages)).toBe(expected);
  });
  test("changing only language makes settings saveable", () => {
    const state: SettingsState = { preferences: { language: "system", menuBarItems: ["disk"], hideDockOnClose: false, updateUseSystemProxy: false }, languageDraft: "system", menuBarDraft: ["disk"], hideDockDraft: false, trayPreview: null, updateProxyDraft: false, preferencesBusy: false, preferencesError: "" };
    expect(preferencesDirty(state)).toBe(false);
    state.languageDraft = "en";
    expect(preferencesDirty(state)).toBe(true);
    state.preferences = { language: "en", menuBarItems: ["disk"], hideDockOnClose: false, updateUseSystemProxy: false };
    expect(preferencesDirty(state)).toBe(false);
    state.hideDockDraft = true;
    expect(preferencesDirty(state)).toBe(true);
    state.hideDockDraft = false;
    state.updateProxyDraft = true;
    expect(preferencesDirty(state)).toBe(true);
  });
  test("switches existing translations without reinitializing the app", async () => {
    await i18n.changeLanguage("zh");
    expect(t("设置")).toBe("设置");
    expect(t("normal")).toBe("正常");
    await i18n.changeLanguage("en");
    expect(t("设置")).toBe("Settings");
    expect(t("normal")).toBe("Normal");
  });
});

test("both catalogs cover UI keys and preserve interpolation contracts", () => {
  expect(Object.keys(en).sort()).toEqual(Object.keys(zh).sort());
  for (const [key, value] of Object.entries(en)) {
    expect(value.trim().length, key).toBeGreaterThan(0);
    expect(value, key).not.toMatch(/[\u4e00-\u9fff]/);
    const placeholders = (text: string) => [...text.matchAll(/{{\w+}}|%[-.\d]*[a-z%]/g)].map(match => match[0]).sort();
    expect(placeholders(value), key).toEqual(placeholders(zh[key as keyof typeof zh]));
  }
  for (const path of ["src/main.ts", "src/views.ts", "src/charts.ts", "src/format.ts", "src/dropdown.ts", "src/settings-view.tsx", "src/menu-bar-view.tsx"]) {
    const source = readFileSync(path, "utf8");
    for (const match of source.matchAll(/\bt\(["']([^"']+)["']/g)) expect(Object.hasOwn(en, match[1]!), `${path}: ${match[1]}`).toBe(true);
  }
});
