// Live metric pages contain generated markup only. Retain their DOM nodes so
// updates do not restart images, replace charts or discard keyboard focus.
// Forms and React-owned pages keep their own update paths.
export function updateLiveContent(root: HTMLElement, html: string): void {
  const template = document.createElement("template");
  template.innerHTML = html;
  patchChildren(root, template.content);
}

function key(node: Node): string | null {
  return node instanceof Element ? node.getAttribute("data-row-key") : null;
}

function patchChildren(current: Node, next: Node): void {
  if (current.nodeName === "TBODY" && key(next.firstChild!)) {
    patchRows(current, next);
    return;
  }
  const keyed = new Map<string, Node>();
  for (const child of current.childNodes) {
    const id = key(child);
    if (id) keyed.set(id, child);
  }
  let cursor = current.firstChild;
  for (const child of Array.from(next.childNodes)) {
    const id = key(child);
    const existing = id ? keyed.get(id) : cursor;
    if (!existing || (id !== key(existing))) {
      current.insertBefore(child, cursor);
      continue;
    }
    if (existing !== cursor) current.insertBefore(existing, cursor);
    cursor = existing.nextSibling;
    patchNode(existing, child);
  }
  while (cursor) { const unused = cursor; cursor = cursor.nextSibling; current.removeChild(unused); }
}

function patchNode(current: Node, next: Node): void {
  if (current.isEqualNode(next)) return;
  if (current.nodeType !== next.nodeType || current.nodeName !== next.nodeName) {
    current.parentNode!.replaceChild(next, current);
  } else if (current instanceof Element && next instanceof Element) {
    for (const attribute of Array.from(current.attributes)) {
      if (!next.hasAttribute(attribute.name)) current.removeAttribute(attribute.name);
    }
    for (const attribute of next.attributes) {
      if (current.getAttribute(attribute.name) !== attribute.value) current.setAttribute(attribute.name, attribute.value);
    }
    patchChildren(current, next);
  } else if (current.nodeValue !== next.nodeValue) current.nodeValue = next.nodeValue;
}

// CPU sorting often rotates groups of rows. Keep the longest ordered sequence
// in place instead of detaching almost the entire table on each sample.
function patchRows(current: Node, next: Node): void {
  const old = Array.from(current.childNodes);
  const rows = Array.from(next.childNodes);
  const byKey = new Map(old.map((node, index) => [key(node), { node, index }]));
  const positions = rows.map(node => byKey.get(key(node))?.index ?? -1);
  const tails: number[] = [], previous: number[] = [];
  for (let i = 0; i < positions.length; i++) {
    if (positions[i]! < 0) continue;
    let low = 0, high = tails.length;
    while (low < high) {
      const middle = (low + high) >>> 1;
      if (positions[tails[middle]!]! < positions[i]!) low = middle + 1; else high = middle;
    }
    previous[i] = low ? tails[low - 1]! : -1;
    tails[low] = i;
  }
  const retained = new Set<number>();
  for (let i = tails.at(-1) ?? -1; i >= 0; i = previous[i]!) retained.add(i);
  const incoming = new Set(rows.map(key));
  for (const node of old) if (!incoming.has(key(node))) current.removeChild(node);
  let anchor: Node | null = null;
  for (let i = rows.length - 1; i >= 0; i--) {
    const nextRow = rows[i]!;
    const row = byKey.get(key(nextRow))?.node ?? nextRow;
    if (!retained.has(i)) current.insertBefore(row, anchor);
    patchNode(row, nextRow);
    anchor = row;
  }
}
