import { createRoot, type Root } from "react-dom/client";
import { createElement } from "react";
import { I18nextProvider } from "react-i18next";
import { SettingsView } from "./settings-view";
import { MenuBarView } from "./menu-bar-view";
import { i18n, type Language } from "./i18n";
import type { ViewState } from "./views";

let root: Root | null = null;
let menuProps: { isMacOS: boolean; refreshSeconds: number } | null = null;

export function renderReactPage(container: HTMLElement, state: ViewState, onLanguage: (language: Language) => void): void {
  if (!root) { container.replaceChildren(); root = createRoot(container); }
  const isMacOS = state.info?.machine.platform === "darwin";
  if (state.page === "menubar") {
    const props = { isMacOS, refreshSeconds: state.info?.refreshSeconds ?? 2 };
    if (props.isMacOS !== menuProps?.isMacOS || props.refreshSeconds !== menuProps?.refreshSeconds) {
      root.render(createElement(I18nextProvider, { i18n }, createElement(MenuBarView, props)));
      menuProps = props;
    }
  } else {
    menuProps = null;
    root.render(createElement(I18nextProvider, { i18n }, createElement(SettingsView, { state, version: state.info?.version ?? "", isMacOS, onLanguage })));
  }
}

export function unmountReactPage(): void {
  root?.unmount(); root = null; menuProps = null;
}
