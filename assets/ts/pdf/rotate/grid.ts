import type { RotatePage, Rotation } from "./types.ts";

const CARD_WIDTH_REM = 9;
const COLUMN_GAP_REM = 0.75;
const ROW_HEIGHT_REM = 14.5;

const OVERSCAN_ROWS = 2;

export interface PDFRotationGrid {
  setPages(previews: string[]): void;

  getRotatedPages(): RotatePage[];

  reset(): void;

  destroy(): void;
}

interface Options {
  root: HTMLElement;

  onChange: (pages: RotatePage[]) => void;
}

export function createPDFRotationGrid(options: Options): PDFRotationGrid {
  const viewport = options.root.querySelector<HTMLElement>("#pdf-rotate-pages");

  const template = options.root.querySelector<HTMLTemplateElement>(
    "#pdf-rotate-page-template",
  );

  if (!viewport || !template) {
    throw new Error("PDF rotation grid workspace is incomplete.");
  }

  return new VirtualPDFRotationGrid(viewport, template, options.onChange);
}

class VirtualPDFRotationGrid implements PDFRotationGrid {
  private readonly viewport: HTMLElement;

  private readonly canvas: HTMLDivElement;

  private readonly template: HTMLTemplateElement;

  private readonly onChange: (pages: RotatePage[]) => void;

  private readonly resizeObserver: ResizeObserver;

  private pages: RotatePage[] = [];

  private columns = 1;

  private cardWidth = 144;

  private columnGap = 12;

  private rowHeight = 232;

  private destroyed = false;

  constructor(
    viewport: HTMLElement,
    template: HTMLTemplateElement,
    onChange: (pages: RotatePage[]) => void,
  ) {
    this.viewport = viewport;

    this.template = template;

    this.onChange = onChange;

    this.canvas = document.createElement("div");

    this.canvas.className = "pdf-rotate-canvas";

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
    this.pages = previews.map((preview, index) => ({
      page: index + 1,

      preview,

      rotation: 0,
    }));

    this.viewport.scrollTop = 0;

    this.emitChange();

    this.updateLayout();
  }

  getRotatedPages(): RotatePage[] {
    return this.pages.filter((page) => page.rotation !== 0);
  }

  reset(): void {
    let changed = false;

    for (const page of this.pages) {
      if (page.rotation === 0) {
        continue;
      }

      page.rotation = 0;

      changed = true;
    }

    if (!changed) {
      return;
    }

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

    this.pages = [];

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

    const rowCount = Math.ceil(this.pages.length / this.columns);

    this.canvas.style.height = `${rowCount * this.rowHeight}px`;

    this.render();
  }

  private render(): void {
    if (this.destroyed || this.pages.length === 0) {
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

    const totalRows = Math.ceil(this.pages.length / this.columns);

    const lastRow = Math.min(totalRows, lastVisibleRow + OVERSCAN_ROWS);

    const startIndex = firstRow * this.columns;

    const endIndex = Math.min(this.pages.length, lastRow * this.columns);

    const fragment = document.createDocumentFragment();

    for (let index = startIndex; index < endIndex; index++) {
      const page = this.pages[index];

      if (!page) {
        continue;
      }

      fragment.append(this.createPageElement(page, index));
    }

    this.canvas.replaceChildren(fragment);
  }

  private createPageElement(page: RotatePage, index: number): HTMLElement {
    const fragment = this.template.content.cloneNode(true);

    if (!(fragment instanceof DocumentFragment)) {
      throw new Error("PDF rotation page template could not be cloned.");
    }

    const item = fragment.querySelector<HTMLElement>(".pdf-rotate-page");

    const number = fragment.querySelector<HTMLElement>(
      ".pdf-rotate-page-number",
    );

    const frame = fragment.querySelector<HTMLElement>(".pdf-rotate-page-frame");

    const image = fragment.querySelector<HTMLImageElement>("img");

    const rotation = fragment.querySelector<HTMLElement>(
      ".pdf-rotate-page-angle",
    );

    const left = fragment.querySelector<HTMLButtonElement>(
      '[data-rotate="left"]',
    );

    const right = fragment.querySelector<HTMLButtonElement>(
      '[data-rotate="right"]',
    );

    if (!item || !number || !frame || !image || !rotation || !left || !right) {
      throw new Error("PDF rotation page template is invalid.");
    }

    number.textContent = String(page.page);

    image.src = page.preview;

    image.alt = `Seite ${page.page}`;

    frame.style.transform = `rotate(${page.rotation}deg)`;

    rotation.textContent =
      page.rotation === 0 ? "Original" : `${page.rotation}°`;

    item.classList.toggle("changed", page.rotation !== 0);

    const row = Math.floor(index / this.columns);

    const column = index % this.columns;

    item.style.width = `${this.cardWidth}px`;

    item.style.transform = `translate(${column * (this.cardWidth + this.columnGap)}px, ${row * this.rowHeight}px)`;

    left.addEventListener("click", () => {
      this.rotate(page.page, -90);
    });

    right.addEventListener("click", () => {
      this.rotate(page.page, 90);
    });

    return item;
  }

  private rotate(pageNumber: number, delta: -90 | 90): void {
    const page = this.pages[pageNumber - 1];

    if (!page) {
      return;
    }

    page.rotation = normalizeRotation(page.rotation + delta);

    this.emitChange();

    this.render();
  }

  private emitChange(): void {
    this.onChange(this.getRotatedPages());
  }
}

function normalizeRotation(value: number): Rotation {
  const normalized = ((value % 360) + 360) % 360;

  switch (normalized) {
    case 90:
      return 90;

    case 180:
      return 180;

    case 270:
      return 270;

    default:
      return 0;
  }
}
