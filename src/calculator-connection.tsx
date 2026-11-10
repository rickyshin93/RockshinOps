import { useLayoutEffect, useRef, useState, type RefObject } from "react";

export type CalculatorTrace = { source: number; target: string; occurrence: number };
type HiddenEndpoint = "source" | "target" | null;
type Endpoints = { source: HTMLElement; target: HTMLElement };

const traceKey = ({ source, target, occurrence }: CalculatorTrace) => `${target}/${source}/${occurrence}`;

function curve(bounds: DOMRect, source: HTMLElement, target: HTMLElement) {
  const from = source.getBoundingClientRect();
  const to = target.getBoundingClientRect();
  const x1 = from.left + from.width / 2 - bounds.left;
  const y1 = from.bottom - bounds.top;
  const x2 = to.left + to.width / 2 - bounds.left;
  const y2 = to.top - bounds.top;
  const bend = (y2 - y1) * .55;
  return `M ${x1} ${y1} C ${x1} ${y1 + bend} ${x2} ${y2 - bend} ${x2} ${y2}`;
}

export function useCalculatorConnections(canvas: RefObject<HTMLDivElement | null>, trace: CalculatorTrace | null) {
  const layer = useRef<SVGSVGElement>(null);
  const syncTrace = useRef<(key: string | null) => void>(undefined);
  const selected = useRef<Endpoints | undefined>(undefined);
  const [hidden, setHidden] = useState<HiddenEndpoint>(null);

  useLayoutEffect(() => {
    const viewport = canvas.current!;
    const svg = layer.current!;
    const list = viewport.querySelector(".calculator-history")!;
    const paths = new Map<string, SVGPathElement>();
    let colored = new Set<HTMLElement>();
    let active: string | null = null;

    function checkViewport() {
      const edge = selected.current;
      const bounds = viewport.getBoundingClientRect();
      const outside = (node: HTMLElement) => {
        const { top, bottom } = node.getBoundingClientRect();
        return bottom > bounds.bottom || top < bounds.top;
      };
      const next = edge ? (outside(edge.target) ? "target" : outside(edge.source) ? "source" : null) : null;
      setHidden(next);
    }

    function measure() {
      const bounds = svg.getBoundingClientRect();
      const byNumber = new Map([...list.querySelectorAll<HTMLElement>("[data-calculation-source]")]
        .map(node => [Number(node.dataset.calculationSource), node]));
      const nextColored = new Set<HTMLElement>();
      const nextPaths = new Set<string>();
      let nextSelected: Endpoints | undefined;

      for (const [index, row] of list.querySelectorAll<HTMLElement>("[data-calculation-row]").entries()) {
        const occurrences = new Map<number, number>();
        for (const target of row.querySelectorAll<HTMLElement>("[data-calculation-link]")) {
          const number = Number(target.dataset.calculationLink!.slice(1));
          const source = byNumber.get(number);
          if (!source) continue;
          const occurrence = occurrences.get(number) ?? 0;
          occurrences.set(number, occurrence + 1);
          const key = traceKey({ source: number, target: row.dataset.calculationRow!, occurrence });
          const tone = String((number - 1) % 5);
          for (const endpoint of [source, target]) {
            if (endpoint.dataset.calculationTone !== tone) endpoint.dataset.calculationTone = tone;
            nextColored.add(endpoint);
          }
          let path = paths.get(key);
          if (!path) {
            path = document.createElementNS("http://www.w3.org/2000/svg", "path");
            path.dataset.calculationTone = tone;
            paths.set(key, path);
            svg.append(path);
          }
          const d = curve(bounds, source, target);
          if (path.getAttribute("d") !== d) path.setAttribute("d", d);
          // The target's zero-based index equals the preceding row's number.
          path.toggleAttribute("data-calculation-adjacent", index === number);
          path.toggleAttribute("data-calculation-active", key === active);
          if (key === active) nextSelected = { source, target };
          nextPaths.add(key);
        }
      }

      for (const [key, path] of paths) {
        if (!nextPaths.has(key)) { path.remove(); paths.delete(key); }
      }
      for (const endpoint of colored) {
        if (!nextColored.has(endpoint)) endpoint.removeAttribute("data-calculation-tone");
      }
      colored = nextColored;

      selected.current = nextSelected;
      checkViewport();
    }

    syncTrace.current = key => { active = key; measure(); };
    const resize = new ResizeObserver(measure);
    resize.observe(list);
    const mutations = new MutationObserver(measure);
    mutations.observe(list, {
      childList: true, subtree: true, characterData: true, attributes: true,
      attributeFilter: ["data-calculation-source", "data-calculation-link", "data-calculation-row"],
    });
    viewport.addEventListener("scroll", checkViewport, { passive: true });
    return () => {
      resize.disconnect();
      mutations.disconnect();
      viewport.removeEventListener("scroll", checkViewport);
      svg.replaceChildren();
      syncTrace.current = undefined;
      selected.current = undefined;
    };
  }, [canvas]);

  useLayoutEffect(() => {
    syncTrace.current?.(trace ? traceKey(trace) : null);
  }, [trace]);

  function jump() {
    if (hidden) selected.current?.[hidden].scrollIntoView({ block: "center" });
  }

  return { layer, hidden, jump };
}
