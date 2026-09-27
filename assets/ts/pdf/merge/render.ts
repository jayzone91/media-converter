import { setupDocumentDrag } from "./sorting.ts";
import type { DocumentRenderActions, MergeDocument } from "./types.ts";

export function renderMergeDocuments(
  root: HTMLElement,
  documents: MergeDocument[],
  actions: DocumentRenderActions,
): void {
  const list = root.querySelector<HTMLElement>("#pdf-merge-list");

  const selection = root.querySelector<HTMLElement>("#pdf-merge-selection");

  const count = root.querySelector<HTMLElement>("#pdf-merge-count");

  const documentTemplate = root.querySelector<HTMLTemplateElement>(
    "#pdf-merge-document-template",
  );

  const pageTemplate = root.querySelector<HTMLTemplateElement>(
    "#pdf-merge-page-template",
  );

  if (!list || !selection || !count || !documentTemplate || !pageTemplate) {
    return;
  }

  list.replaceChildren();

  selection.hidden = documents.length === 0;

  count.textContent =
    documents.length === 1 ? "1 Datei" : `${documents.length} Dateien`;

  documents.forEach((document, index) => {
    list.append(
      createDocumentElement(
        root,
        documentTemplate,
        pageTemplate,
        document,
        index,
        documents.length,
        actions,
      ),
    );
  });
}

function createDocumentElement(
  root: HTMLElement,
  documentTemplate: HTMLTemplateElement,
  pageTemplate: HTMLTemplateElement,
  document: MergeDocument,
  index: number,
  documentCount: number,
  actions: DocumentRenderActions,
): HTMLElement {
  const fragment = documentTemplate.content.cloneNode(true);

  if (!(fragment instanceof DocumentFragment)) {
    throw new Error("PDF document template could not be cloned.");
  }

  const item = fragment.querySelector<HTMLElement>(".pdf-file-item");

  if (!item) {
    throw new Error("PDF document template is invalid.");
  }

  item.dataset.uploadId = document.id;

  const position = item.querySelector<HTMLElement>(".pdf-file-index");

  if (position) {
    position.textContent = String(index + 1);
  }

  const filename = item.querySelector<HTMLElement>(".pdf-file-name");

  if (filename) {
    filename.textContent = document.filename;

    filename.title = document.filename;
  }

  const meta = item.querySelector<HTMLElement>(".pdf-file-meta");

  if (meta) {
    meta.textContent = formatDocumentMeta(document);
  }

  const pageStrip = item.querySelector<HTMLElement>(".pdf-page-strip");

  if (pageStrip) {
    renderPages(pageStrip, pageTemplate, document);
  }

  const moveUp = item.querySelector<HTMLButtonElement>(
    '[data-action="move-up"]',
  );

  const moveDown = item.querySelector<HTMLButtonElement>(
    '[data-action="move-down"]',
  );

  const remove = item.querySelector<HTMLButtonElement>(
    '[data-action="remove"]',
  );

  if (moveUp) {
    moveUp.disabled = index === 0;

    moveUp.addEventListener("click", () => {
      actions.moveUp(document.id);
    });
  }

  if (moveDown) {
    moveDown.disabled = index === documentCount - 1;

    moveDown.addEventListener("click", () => {
      actions.moveDown(document.id);
    });
  }

  remove?.addEventListener("click", () => {
    actions.remove(document.id);
  });

  setupDocumentDrag(item, document.id, root, actions.reorder);

  return item;
}

function renderPages(
  container: HTMLElement,
  template: HTMLTemplateElement,
  document: MergeDocument,
): void {
  document.previews.forEach((preview, index) => {
    const fragment = template.content.cloneNode(true);

    if (!(fragment instanceof DocumentFragment)) {
      return;
    }

    const figure = fragment.querySelector<HTMLElement>(".pdf-page-preview");

    const image = fragment.querySelector<HTMLImageElement>("img");

    const caption = fragment.querySelector<HTMLElement>("figcaption");

    if (!figure || !image || !caption) {
      return;
    }

    const pageNumber = index + 1;

    figure.dataset.page = String(pageNumber);

    image.src = preview;

    image.alt = `${document.filename}, Seite ${pageNumber}`;

    caption.textContent = `Seite ${pageNumber}`;

    container.append(fragment);
  });
}

function formatDocumentMeta(document: MergeDocument): string {
  const pages =
    document.pageCount === 1 ? "1 Seite" : `${document.pageCount} Seiten`;

  return `${pages} · ${formatBytes(document.size)}`;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) {
    return "0 B";
  }

  const units = ["B", "KiB", "MiB", "GiB"] as const;

  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );

  const unit = units[index] ?? "B";

  const value = bytes / 1024 ** index;

  return `${value.toFixed(index === 0 ? 0 : 1)} ${unit}`;
}
