import type { SortPage } from "./types.ts";

export function renderSortPages(
  root: HTMLElement,
  pages: SortPage[],
  reorder: (sourcePage: number, targetPage: number) => void,
): void {
  const container = root.querySelector<HTMLElement>("#pdf-sort-pages");

  const template = root.querySelector<HTMLTemplateElement>(
    "#pdf-sort-page-template",
  );

  if (!container || !template) {
    return;
  }

  container.replaceChildren();

  pages.forEach((page, index) => {
    container.append(createPageElement(root, template, page, index, reorder));
  });
}

function createPageElement(
  root: HTMLElement,
  template: HTMLTemplateElement,
  page: SortPage,
  index: number,
  reorder: (sourcePage: number, targetPage: number) => void,
): HTMLElement {
  const fragment = template.content.cloneNode(true);

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

  if (!item || !position || !image || !original) {
    throw new Error("PDF sort page template is invalid.");
  }

  item.dataset.page = String(page.originalPage);

  position.textContent = String(index + 1);

  image.src = page.preview;

  image.alt = `Originalseite ${page.originalPage}`;

  original.textContent = `Original ${page.originalPage}`;

  setupPageDrag(root, item, page.originalPage, reorder);

  return item;
}

function setupPageDrag(
  root: HTMLElement,
  item: HTMLElement,
  page: number,
  reorder: (sourcePage: number, targetPage: number) => void,
): void {
  item.addEventListener("dragstart", (event: DragEvent) => {
    item.classList.add("dragging");

    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";

      event.dataTransfer.setData("application/x-pdf-page", String(page));
    }
  });

  item.addEventListener("dragend", () => {
    item.classList.remove("dragging");

    clearTargets(root);
  });

  item.addEventListener("dragover", (event: DragEvent) => {
    event.preventDefault();

    const source = readDraggedPage(event);

    if (source === null || source === page) {
      return;
    }

    clearTargets(root);

    item.classList.add("drag-target");

    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
  });

  item.addEventListener("dragleave", () => {
    item.classList.remove("drag-target");
  });

  item.addEventListener("drop", (event: DragEvent) => {
    event.preventDefault();

    item.classList.remove("drag-target");

    const source = readDraggedPage(event);

    if (source === null || source === page) {
      return;
    }

    reorder(source, page);
  });
}

function readDraggedPage(event: DragEvent): number | null {
  const value = event.dataTransfer?.getData("application/x-pdf-page");

  if (!value) {
    return null;
  }

  const page = Number.parseInt(value, 10);

  return Number.isInteger(page) ? page : null;
}

function clearTargets(root: HTMLElement): void {
  root
    .querySelectorAll<HTMLElement>(".pdf-sort-page.drag-target")
    .forEach((item) => {
      item.classList.remove("drag-target");
    });
}
