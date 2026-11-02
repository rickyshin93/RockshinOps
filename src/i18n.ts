import { createInstance } from "i18next";
import { initReactI18next } from "react-i18next";
import en from "../internal/localization/en.json";
import zh from "../internal/localization/zh.json";

export type Language = "system" | "en" | "zh";
export { resolveLanguage } from "./i18n-language";
export const i18n = createInstance();
void i18n.use(initReactI18next).init({
  resources: { en: { translation: en }, zh: { translation: zh } },
  lng: "en",
  fallbackLng: "en", supportedLngs: ["en", "zh"], initAsync: false,
  keySeparator: false, nsSeparator: false, interpolation: { escapeValue: false },
});
export function t(key: string, values?: Record<string, unknown>): string { return i18n.t(key, values); }
