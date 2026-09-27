const CARD_WIDTH_REM = 9;
const COLUMN_GAP_REM = 0.75;
const ROW_HEIGHT_REM = 14.5;

const OVERSCAN_ROWS = 2;
const MAX_SPLIT_POINTS = 199;

export interface PDFSplitGrid {
  setPages(previews: string[]): void;

  getSplitPoints(): number[];

  clear(): void;

  destroy(): void;
}

interface Options {
  root: HTMLElement;

  onChange: (splitPoints: number[]) => void;

  onLimitReached: () => void;
}

export function createPDFSplitGrid(options: Options): PDFSplitGrid {
  const viewport = options.root.querySelector<HTMLElement>("#pdf-split-pages");

  const template = options.root.querySelector<HTMLTemplateElement>(
    "#pdf-split-page-template",
  );

  if (!viewport || !template) {
    throw new Error("PDF split grid workspace is incomplete.");
  }

  return new VirtualPDFSplitGrid(
    viewport,
    template,
    options.onChange,
    options.onLimitReached,
  );
}

class VirtualPDFSplitGrid implements PDFSplitGrid {
  private readonly viewport: HTMLElement;

  private readonly canvas: HTMLDivElement;

  private readonly template: HTMLTemplateElement;

  private readonly onChange: (splitPoints: number[]) => void;

  private readonly onLimitReached: () => void;

  private readonly resizeObserver: ResizeObserver;

  private previews: string[] = [];

  private readonly splitPoints = new Set<number>();

  private columns = 1;

  private cardWidth = 144;
  private columnGap = 12;
  private rowHeight = 232;

  private destroyed = false;

  constructor(
    viewport: HTMLElement,
    template: HTMLTemplateElement,
    onChange: (splitPoints: number[]) => void,
    onLimitReached: () => void,
  ) {
    this.viewport = viewport;

    this.template = template;

    this.onChange = onChange;

    this.onLimitReached = onLimitReached;

    this.canvas = document.createElement("div");

    this.canvas.className = "pdf-split-canvas";

    this.viewport.replaceChildren(this.canvas);

    this.updateDimensions();

    this.resizeObserver = new ResizeObserver(() => {
      this.updateLayout();
    });

    this.resizeObserver.observe(this.viewport);

    this.viewport.addEventListener("scroll", this.handleScroll, {
      passive: true,
    });

    this.updateLayout();
  }

  setPages(previews: string[]): void {
    this.previews = previews;

    this.splitPoints.clear();

    this.viewport.scrollTop = 0;

    this.emitChange();

    this.updateLayout();
  }

  getSplitPoints(): number[] {
    return Array.from(this.splitPoints).sort((left, right) => left - right);
  }

  clear(): void {
    if (this.splitPoints.size === 0) {
      return;
    }

    this.splitPoints.clear();

    this.emitChange();

    this.render();
  }

  destroy(): void {
    if (this.destroyed) {
      return;
    }

    this.destroyed = true;

    this.resizeObserver.disconnect();

    this.viewport.removeEventListener("scroll", this.handleScroll);

    this.previews = [];

    this.splitPoints.clear();

    this.canvas.replaceChildren();
  }

  private readonly handleScroll = (): void => {
    this.render();
  };

  private updateDimensions(): void {
    const rootFontSize =
      Number.parseFloat(getComputedStyle(document.documentElement).fontSize) ||
      16;

    this.cardWidth = CARD_WIDTH_REM * rootFontSize;

    this.columnGap = COLUMN_GAP_REM * rootFontSize;

    this.rowHeight = ROW_HEIGHT_REM * rootFontSize;
  }

  private updateLayout(): void {
    if (this.destroyed) {
      return;
    }

    this.updateDimensions();

    const slotWidth = this.cardWidth + this.columnGap;

    this.columns = Math.max(
      1,
      Math.floor((this.viewport.clientWidth + this.columnGap) / slotWidth),
    );

    const rowCount = Math.ceil(this.previews.length / this.columns);

    this.canvas.style.height = `${rowCount * this.rowHeight}px`;

    this.render();
  }

  private render(): void {
    if (this.destroyed || this.previews.length === 0) {
      this.canvas.replaceChildren();

      return;
    }

    const firstVisibleRow = Math.floor(
      this.viewport.scrollTop / this.rowHeight,
    );

    const lastVisibleRow = Math.ceil(
      (this.viewport.scrollTop + this.viewport.clientHeight) / this.rowHeight,
    );

    const firstRow = Math.max(0, firstVisibleRow - OVERSCAN_ROWS);

    const totalRows = Math.ceil(this.previews.length / this.columns);

    const lastRow = Math.min(totalRows, lastVisibleRow + OVERSCAN_ROWS);

    const startIndex = firstRow * this.columns;

    const endIndex = Math.min(this.previews.length, lastRow * this.columns);

    const fragment = document.createDocumentFragment();

    for (let index = startIndex; index < endIndex; index++) {
      const preview = this.previews[index];

      if (!preview) {
        continue;
      }

      fragment.append(this.createPageElement(preview, index));
    }

    this.canvas.replaceChildren(fragment);
  }

  private createPageElement(preview: string, index: number): HTMLElement {
    const fragment = this.template.content.cloneNode(true);

    if (!(fragment instanceof DocumentFragment)) {
      throw new Error("PDF split page template could not be cloned.");
    }

    const item = fragment.querySelector<HTMLElement>(".pdf-split-page");

    const number = fragment.querySelector<HTMLElement>(
      ".pdf-split-page-number",
    );

    const image = fragment.querySelector<HTMLImageElement>("img");

    const button =
      fragment.querySelector<HTMLButtonElement>(".pdf-split-toggle");

    const label = fragment.querySelector<HTMLElement>(
      ".pdf-split-toggle-label",
    );

    if (!item || !number || !image || !button || !label) {
      throw new Error("PDF split page template is invalid.");
    }

    const page = index + 1;

    const active = this.splitPoints.has(page);

    number.textContent = String(page);

    image.src = preview;

    image.alt = `Seite ${page}`;

    item.classList.toggle("split-after", active);

    button.classList.toggle("active", active);

    button.setAttribute("aria-pressed", active ? "true" : "false");

    if (page === this.previews.length) {
      button.hidden = true;
    } else {
      label.textContent = active
        ? `Trennung nach Seite ${page}`
        : `Nach Seite ${page} trennen`;
    }

    const row = Math.floor(index / this.columns);

    const column = index % this.columns;

    item.style.width = `${this.cardWidth}px`;

    item.style.transform = `translate(${column * (this.cardWidth + this.columnGap)}px, ${row * this.rowHeight}px)`;

    button.addEventListener("click", () => {
      this.toggleSplitPoint(page);
    });

    return item;
  }

  private toggleSplitPoint(page: number): void {
    if (page < 1 || page >= this.previews.length) {
      return;
    }

    if (this.splitPoints.has(page)) {
      this.splitPoints.delete(page);
    } else {
      if (this.splitPoints.size >= MAX_SPLIT_POINTS) {
        this.onLimitReached();

        return;
      }

      this.splitPoints.add(page);
    }

    this.emitChange();

    this.render();
  }

  private emitChange(): void {
    this.onChange(this.getSplitPoints());
  }
}
