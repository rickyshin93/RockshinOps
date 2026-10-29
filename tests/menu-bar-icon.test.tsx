import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { MenuBarIcon } from "../src/menu-bar-icon";

test("missing captures show an icon placeholder instead of app-name text", () => {
  const html = renderToStaticMarkup(<MenuBarIcon image="" name="Flux" />);
  expect(html).toContain("<svg");
  expect(html).not.toContain("Flux");
  expect(html).not.toContain("<img");
});

test("successful captures render the real menu-bar image with its accessible name", () => {
  const html = renderToStaticMarkup(<MenuBarIcon image="data:image/png;base64,captured" name="Flux" />);
  expect(html).toContain('src="data:image/png;base64,captured"');
  expect(html).toContain('alt="Flux"');
  expect(html).not.toContain("<svg");
});
