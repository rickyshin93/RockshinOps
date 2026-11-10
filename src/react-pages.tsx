import { createRoot, type Root } from "react-dom/client";
import { createElement } from "react";
import { I18nextProvider } from "react-i18next";
import { LauncherSettingsView } from "./launcher-settings-view";
import { CalculatorView, type CalculatorImport } from "./calculator-view";
import { SettingsView } from "./settings-view";
import { MenuBarView } from "./menu-bar-view";
import { SpaceView } from "./space-view";
import { i18n, type Language } from "./i18n";
import type { ViewState } from "./views";

let root: Root | null = null;
let calculatorOwner: Root | null = null;
let menuProps: { isMacOS: boolean; refreshSeconds: number } | null = null;
export type CalculatorDraft = { expression: string; rowIDs?: readonly string[]; incoming?: CalculatorImport; finishImport: (request: CalculatorImport) => void };

export function renderReactPage(container: HTMLElement, state: ViewState, onLanguage: (language: Language) => void, openDiskAccess: () => void, onLauncherShortcut: (shortcut: string) => void, calculatorDraft: CalculatorDraft): void {
  if (!root) { container.replaceChildren(); root = createRoot(container); }
  const isMacOS = state.info?.machine.platform === "darwin";
  if (state.page === "menubar") {
    const props = { isMacOS, refreshSeconds: state.info?.refreshSeconds ?? 2 };
    if (props.isMacOS !== menuProps?.isMacOS || props.refreshSeconds !== menuProps?.refreshSeconds) {
      root.render(createElement(I18nextProvider, { i18n }, createElement(MenuBarView, props)));
      menuProps = props;
    }
  } else if (state.page === "cleanup") {
    menuProps = null;
    const disk = state.snapshot?.disk;
    if (disk) root.render(createElement(I18nextProvider, { i18n }, createElement(SpaceView, { volume: disk.volume, capacityError: disk.capacityError, isMacOS, openDiskAccess })));
  } else if (state.page === "launcher") {
    menuProps = null;
    root.render(createElement(I18nextProvider, { i18n }, createElement(LauncherSettingsView, { state, isMacOS, onLauncherShortcut })));
  } else if (state.page === "calculator") {
    menuProps = null;
    const owner = root;
    calculatorOwner = owner;
    root.render(createElement(I18nextProvider, { i18n }, createElement(CalculatorView, {
      initialExpression: calculatorDraft.expression,
      initialRowIDs: calculatorDraft.rowIDs,
      incoming: calculatorDraft.incoming,
      onImportHandled: calculatorDraft.finishImport,
      onEdit: (expression, rowIDs) => {
        if (calculatorOwner !== owner) return;
        calculatorDraft.expression = expression;
        calculatorDraft.rowIDs = expression.includes("#") ? rowIDs : undefined;
      },
    })));
  } else {
    menuProps = null;
    root.render(createElement(I18nextProvider, { i18n }, createElement(SettingsView, { state, version: state.info?.version ?? "", isMacOS, onLanguage })));
  }
}

export function unmountReactPage(): void {
  root?.unmount(); root = null; menuProps = null;
}
