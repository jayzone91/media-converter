const CARD_WIDTH_REM = 9;
const COLUMN_GAP_REM = 0.75;
const ROW_HEIGHT_REM = 13.75;

const OVERSCAN_ROWS = 2;

export interface PDFSelectionGrid {
  setPages(previews: string[]): void;

  getSelectedPages(): number[];

  selectAll(): void;

  clearSelection(): void;

  destroy(): void;
}

interface Options {
  root: HTMLElement;

  viewportSelector: string;

  templateSelector: string;

  onSelectionChange: (selectedPages: number[]) => void;
}

export function createPDFSelectionGrid(options: Options): PDFSelectionGrid {
  const viewport = options.root.querySelector<HTMLElement>(
    options.viewportSelector,
  );

  const template = options.root.querySelector<HTMLTemplateElement>(
    options.templateSelector,
  );

  if (!viewport || !template) {
    throw new Error("PDF selection grid workspace is incomplete.");
  }

  return new VirtualPDFSelectionGrid(
    viewport,
    template,
    options.onSelectionChange,
  );
}

class VirtualPDFSelectionGrid implements PDFSelectionGrid {
  private readonly viewport: HTMLElement;

  private readonly canvas: HTMLDivElement;

  private readonly template: HTMLTemplateElement;

  private readonly onSelectionChange: (selectedPages: number[]) => void;

  private readonly resizeObserver: ResizeObserver;

  private previews: string[] = [];

  private readonly selected = new Set<number>();

  private lastSelectedPage: number | null = null;

  private columns = 1;

  private cardWidth = 144;

  private columnGap = 12;

  private rowHeight = 220;

  private destroyed = false;

  constructor(
    viewport: HTMLElement,
    template: HTMLTemplateElement,
    onSelectionChange: (selectedPages: number[]) => void,
  ) {
    this.viewport = viewport;

    this.template = template;

    this.onSelectionChange = onSelectionChange;

    this.canvas = document.createElement("div");

    this.canvas.className = "pdf-page-selector-canvas";

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

    this.selected.clear();

    this.lastSelectedPage = null;

    this.viewport.scrollTop = 0;

    this.emitSelection();

    this.updateLayout();
  }

  getSelectedPages(): number[] {
    return Array.from(this.selected).sort((left, right) => left - right);
  }

  selectAll(): void {
    this.selected.clear();

    for (let page = 1; page <= this.previews.length; page++) {
      this.selected.add(page);
    }

    this.lastSelectedPage =
      this.previews.length > 0 ? this.previews.length : null;

    this.emitSelection();

    this.render();
  }

  clearSelection(): void {
    if (this.selected.size === 0) {
      return;
    }

    this.selected.clear();

    this.lastSelectedPage = null;

    this.emitSelection();

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

    this.selected.clear();

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

    const availableWidth = this.viewport.clientWidth;

    const slotWidth = this.cardWidth + this.columnGap;

    this.columns = Math.max(
      1,
      Math.floor((availableWidth + this.columnGap) / slotWidth),
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

    const scrollTop = this.viewport.scrollTop;

    const viewportHeight = this.viewport.clientHeight;

    const firstVisibleRow = Math.floor(scrollTop / this.rowHeight);

    const lastVisibleRow = Math.ceil(
      (scrollTop + viewportHeight) / this.rowHeight,
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
      throw new Error("PDF selection page template could not be cloned.");
    }

    const item = fragment.querySelector<HTMLButtonElement>(".pdf-select-page");

    const number = fragment.querySelector<HTMLElement>(
      ".pdf-select-page-number",
    );

    const image = fragment.querySelector<HTMLImageElement>("img");

    const footer = fragment.querySelector<HTMLElement>(
      ".pdf-select-page-footer",
    );

    if (!item || !number || !image || !footer) {
      throw new Error("PDF selection page template is invalid.");
    }

    const page = index + 1;

    const selected = this.selected.has(page);

    item.dataset.page = String(page);

    item.classList.toggle("selected", selected);

    item.setAttribute("aria-pressed", selected ? "true" : "false");

    number.textContent = String(page);

    image.src = preview;

    image.alt = `Seite ${page}`;

    footer.textContent = `Seite ${page}`;

    const row = Math.floor(index / this.columns);

    const column = index % this.columns;

    item.style.width = `${this.cardWidth}px`;

    item.style.transform = `translate(${column * (this.cardWidth + this.columnGap)}px, ${row * this.rowHeight}px)`;

    item.addEventListener("click", (event: MouseEvent) => {
      this.togglePage(page, event.shiftKey);
    });

    return item;
  }

  private togglePage(page: number, rangeSelection: boolean): void {
    if (rangeSelection && this.lastSelectedPage !== null) {
      const start = Math.min(page, this.lastSelectedPage);

      const end = Math.max(page, this.lastSelectedPage);

      for (let current = start; current <= end; current++) {
        this.selected.add(current);
      }
    } else if (this.selected.has(page)) {
      this.selected.delete(page);
    } else {
      this.selected.add(page);
    }

    this.lastSelectedPage = page;

    this.emitSelection();

    this.render();
  }

  private emitSelection(): void {
    this.onSelectionChange(this.getSelectedPages());
  }
}
