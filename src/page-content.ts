// Only the two large inspection tables retain detached DOM between visits.
// Other pages (including React forms) keep their existing mount lifecycle.
export class PageContent {
  page: string | null = null;
  private cached = new Map<string, { nodes: Node[]; html: string }>();

  constructor(private root: HTMLElement) {}

  activate(page: string, html: string): string {
    if (page === this.page) return html;
    if (this.page === "processes" || this.page === "connections") {
      this.cached.set(this.page, { nodes: Array.from(this.root.childNodes), html });
    }
    const saved = this.cached.get(page);
    this.cached.delete(page);
    this.root.replaceChildren(...(saved?.nodes ?? []));
    this.page = page;
    return saved?.html ?? "";
  }
}
