import type { SortPage } from "./types.ts";

const CARD_WIDTH_REM = 9;
const COLUMN_GAP_REM = 0.75;
const ROW_HEIGHT_REM = 14.5;

const OVERSCAN_ROWS = 2;

const AUTO_SCROLL_EDGE = 72;
const AUTO_SCROLL_SPEED = 18;

interface DragState {
  sourcePage: number;
  sourceIndex: number;
  targetIndex: number;

  clientX: number;
  clientY: number;
}

export interface SortPageRenderer {
  setPages(pages: SortPage[]): void;

  destroy(): void;
}

export function createSortPageRenderer(
  root: HTMLElement,
  reorder: (sourcePage: number, targetPage: number) => void,
): SortPageRenderer {
  const viewport = root.querySelector<HTMLElement>("#pdf-sort-pages");

  const template = root.querySelector<HTMLTemplateElement>(
    "#pdf-sort-page-template",
  );

  if (!viewport || !template) {
    throw new Error("PDF sort workspace is incomplete.");
  }

  return new VirtualSortPageRenderer(viewport, template, reorder);
}

class VirtualSortPageRenderer implements SortPageRenderer {
  private readonly viewport: HTMLElement;

  private readonly canvas: HTMLDivElement;

  private readonly template: HTMLTemplateElement;

  private readonly reorder: (sourcePage: number, targetPage: number) => void;

  private readonly resizeObserver: ResizeObserver;

  private pages: SortPage[] = [];

  private columns = 1;

  private cardWidth = 144;
  private columnGap = 12;
  private rowHeight = 232;

  private drag: DragState | null = null;

  private autoScrollFrame: number | null = null;

  private destroyed = false;

  constructor(
    viewport: HTMLElement,
    template: HTMLTemplateElement,
    reorder: (sourcePage: number, targetPage: number) => void,
  ) {
    this.viewport = viewport;

    this.template = template;

    this.reorder = reorder;

    this.canvas = document.createElement("div");

    this.canvas.className = "pdf-sort-canvas";

    this.viewport.replaceChildren(this.canvas);

    this.updateDimensions();

    this.resizeObserver = new ResizeObserver(() => {
      this.updateLayout();
    });

    this.resizeObserver.observe(this.viewport);

    this.viewport.addEventListener("scroll", this.handleScroll, {
      passive: true,
    });

    window.addEventListener("pointermove", this.handlePointerMove);

    window.addEventListener("pointerup", this.handlePointerUp);

    window.addEventListener("pointercancel", this.handlePointerCancel);

    this.updateLayout();
  }

  setPages(pages: SortPage[]): void {
    this.pages = pages;

    this.updateLayout();
  }

  destroy(): void {
    if (this.destroyed) {
      return;
    }

    this.destroyed = true;

    this.stopAutoScroll();

    this.resizeObserver.disconnect();

    this.viewport.removeEventListener("scroll", this.handleScroll);

    window.removeEventListener("pointermove", this.handlePointerMove);

    window.removeEventListener("pointerup", this.handlePointerUp);

    window.removeEventListener("pointercancel", this.handlePointerCancel);

    document.body.classList.remove("pdf-sort-dragging");

    this.drag = null;

    this.pages = [];

    this.canvas.replaceChildren();
  }

  private readonly handleScroll = (): void => {
    this.render();
  };

  private readonly handlePointerMove = (event: PointerEvent): void => {
    if (!this.drag || this.destroyed) {
      return;
    }

    event.preventDefault();

    this.drag.clientX = event.clientX;

    this.drag.clientY = event.clientY;

    this.updateDragTarget();

    this.ensureAutoScroll();
  };

  private readonly handlePointerUp = (): void => {
    this.finishDrag(true);
  };

  private readonly handlePointerCancel = (): void => {
    this.finishDrag(false);
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

    const rowCount = Math.ceil(this.pages.length / this.columns);

    this.canvas.style.height = `${rowCount * this.rowHeight}px`;

    this.render();
  }

  private render(): void {
    if (this.destroyed || this.pages.length === 0) {
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

  private createPageElement(page: SortPage, index: number): HTMLElement {
    const fragment = this.template.content.cloneNode(true);

    if (!(fragment instanceof DocumentFragment)) {
      throw new Error("PDF sort page template could not be cloned.");
    }

    const item = fragment.querySelector<HTMLElement>(".pdf-sort-page");

    const position = fragment.querySelector<HTMLElement>(
      ".pdf-sort-page-position",
    );

    const image = fragment.querySelector<HTMLImageElement>("img");

    const original = fragment.querySelector<HTMLElement>(
      ".pdf-sort-page-original",
    );

    const handle = fragment.querySelector<HTMLElement>(".pdf-sort-page-drag");

    if (!item || !position || !image || !original || !handle) {
      throw new Error("PDF sort page template is invalid.");
    }

    /*
     * Wir verwenden kein natives HTML5 Drag & Drop mehr.
     * Der Pointer-Handler funktioniert auch während
     * Virtualisierung und Auto-Scroll.
     */
    item.draggable = false;

    item.dataset.page = String(page.originalPage);

    position.textContent = String(index + 1);

    image.src = page.preview;

    image.alt = `Originalseite ${page.originalPage}`;

    original.textContent = `Original ${page.originalPage}`;

    const row = Math.floor(index / this.columns);

    const column = index % this.columns;

    item.style.width = `${this.cardWidth}px`;

    item.style.transform = `translate(${column * (this.cardWidth + this.columnGap)}px, ${row * this.rowHeight}px)`;

    if (this.drag?.sourcePage === page.originalPage) {
      item.classList.add("dragging");
    }

    if (this.drag?.targetIndex === index && this.drag.sourceIndex !== index) {
      item.classList.add("drag-target");
    }

    handle.addEventListener("pointerdown", (event: PointerEvent) => {
      this.startDrag(event, page.originalPage, index);
    });

    return item;
  }

  private startDrag(event: PointerEvent, page: number, index: number): void {
    if (this.drag || event.button !== 0) {
      return;
    }

    event.preventDefault();

    this.drag = {
      sourcePage: page,

      sourceIndex: index,

      targetIndex: index,

      clientX: event.clientX,

      clientY: event.clientY,
    };

    document.body.classList.add("pdf-sort-dragging");

    this.render();

    this.ensureAutoScroll();
  }

  private updateDragTarget(): void {
    if (!this.drag) {
      return;
    }

    const targetIndex = this.getIndexFromPointer(
      this.drag.clientX,
      this.drag.clientY,
    );

    if (targetIndex === this.drag.targetIndex) {
      return;
    }

    this.drag.targetIndex = targetIndex;

    this.render();
  }

  private getIndexFromPointer(clientX: number, clientY: number): number {
    const rect = this.viewport.getBoundingClientRect();

    const localX = Math.max(0, clientX - rect.left);

    const localY = Math.max(0, clientY - rect.top + this.viewport.scrollTop);

    const slotWidth = this.cardWidth + this.columnGap;

    const column = Math.min(
      this.columns - 1,
      Math.max(0, Math.floor(localX / slotWidth)),
    );

    const row = Math.max(0, Math.floor(localY / this.rowHeight));

    return Math.min(this.pages.length - 1, row * this.columns + column);
  }

  private ensureAutoScroll(): void {
    if (!this.drag || this.autoScrollFrame !== null) {
      return;
    }

    this.autoScrollFrame = requestAnimationFrame(this.autoScroll);
  }

  private readonly autoScroll = (): void => {
    this.autoScrollFrame = null;

    if (!this.drag || this.destroyed) {
      return;
    }

    const rect = this.viewport.getBoundingClientRect();

    const distanceTop = this.drag.clientY - rect.top;

    const distanceBottom = rect.bottom - this.drag.clientY;

    let delta = 0;

    if (distanceTop < AUTO_SCROLL_EDGE) {
      const factor = 1 - Math.max(0, distanceTop) / AUTO_SCROLL_EDGE;

      delta = -AUTO_SCROLL_SPEED * factor;
    } else if (distanceBottom < AUTO_SCROLL_EDGE) {
      const factor = 1 - Math.max(0, distanceBottom) / AUTO_SCROLL_EDGE;

      delta = AUTO_SCROLL_SPEED * factor;
    }

    if (delta !== 0) {
      const before = this.viewport.scrollTop;

      this.viewport.scrollTop += delta;

      if (this.viewport.scrollTop !== before) {
        this.updateDragTarget();
      }
    }

    this.autoScrollFrame = requestAnimationFrame(this.autoScroll);
  };

  private stopAutoScroll(): void {
    if (this.autoScrollFrame === null) {
      return;
    }

    cancelAnimationFrame(this.autoScrollFrame);

    this.autoScrollFrame = null;
  }

  private finishDrag(apply: boolean): void {
    if (!this.drag) {
      return;
    }

    const drag = this.drag;

    this.drag = null;

    this.stopAutoScroll();

    document.body.classList.remove("pdf-sort-dragging");

    if (apply && drag.targetIndex !== drag.sourceIndex) {
      const target = this.pages[drag.targetIndex];

      if (target) {
        this.reorder(drag.sourcePage, target.originalPage);

        return;
      }
    }

    this.render();
  }
}
