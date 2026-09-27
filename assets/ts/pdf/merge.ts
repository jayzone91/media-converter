import { downloadBlob, getDownloadFilename } from "../shared/download.ts";

interface PDFUploadResponse {
  id: string;
  filename: string;
  size: number;
  page_count: number;
  previews: string[];
}

interface MergeDocument {
  id: string;
  filename: string;
  size: number;
  pageCount: number;
  previews: string[];
}

const MAX_FILES = 50;

const MAX_FILE_SIZE = 512 * 1024 * 1024;

const MAX_TOTAL_SIZE = 1024 * 1024 * 1024;

let root: HTMLElement | null = null;

let documents: MergeDocument[] = [];

let uploading = false;
let merging = false;

export function setupPDFMerge(workspace: HTMLElement): void {
  root = workspace;
  documents = [];
  uploading = false;
  merging = false;

  const input = getElement<HTMLInputElement>("#pdf-merge-files");

  const dropZone = getElement<HTMLElement>("#pdf-merge-drop-zone");

  const clearButton = getElement<HTMLButtonElement>("#pdf-merge-clear");

  const submitButton = getElement<HTMLButtonElement>("#pdf-merge-submit");

  if (!input || !dropZone || !clearButton || !submitButton) {
    return;
  }

  input.addEventListener("change", () => {
    const files = input.files ? Array.from(input.files) : [];

    input.value = "";

    if (files.length > 0) {
      void addFiles(files);
    }
  });

  setupDropZone(dropZone);

  clearButton.addEventListener("click", () => {
    void clearDocuments();
  });

  submitButton.addEventListener("click", () => {
    void mergeDocuments();
  });

  renderDocuments();
}

export async function destroyPDFMerge(): Promise<void> {
  const ids = documents.map((document) => document.id);

  documents = [];
  root = null;

  await Promise.allSettled(ids.map(deleteUpload));
}

function setupDropZone(dropZone: HTMLElement): void {
  const events = ["dragenter", "dragover", "dragleave", "drop"] as const;

  for (const eventName of events) {
    dropZone.addEventListener(eventName, (event) => {
      event.preventDefault();
      event.stopPropagation();
    });
  }

  for (const eventName of ["dragenter", "dragover"] as const) {
    dropZone.addEventListener(eventName, () => {
      dropZone.classList.add("drag-over");
    });
  }

  for (const eventName of ["dragleave", "drop"] as const) {
    dropZone.addEventListener(eventName, () => {
      dropZone.classList.remove("drag-over");
    });
  }

  dropZone.addEventListener("drop", (event: DragEvent) => {
    const files = event.dataTransfer?.files;

    if (!files?.length) {
      return;
    }

    void addFiles(Array.from(files));
  });
}

async function addFiles(files: File[]): Promise<void> {
  if (uploading || merging) {
    return;
  }

  clearError();

  const validFiles = validateFiles(files);

  if (validFiles.length === 0) {
    return;
  }

  uploading = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    for (let index = 0; index < validFiles.length; index++) {
      const file = validFiles[index];

      if (!file) {
        continue;
      }

      updateUploadProgress(file.name, index + 1, validFiles.length);

      try {
        const upload = await uploadPDF(file);

        documents.push({
          id: upload.id,

          filename: upload.filename,

          size: upload.size,

          pageCount: upload.page_count,

          previews: upload.previews,
        });

        renderDocuments();
      } catch (error: unknown) {
        showError(
          error instanceof Error
            ? `${file.name}: ${error.message}`
            : `${file.name}: Upload fehlgeschlagen.`,
        );
      }
    }
  } finally {
    uploading = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

function validateFiles(files: File[]): File[] {
  const result: File[] = [];

  let totalSize = getTotalSize();

  for (const file of files) {
    const validType =
      file.type === "application/pdf" ||
      file.name.toLowerCase().endsWith(".pdf");

    if (!validType) {
      showError(`${file.name}: Es können nur PDF-Dateien hinzugefügt werden.`);

      continue;
    }

    if (file.size <= 0) {
      showError(`${file.name}: Die Datei ist leer.`);

      continue;
    }

    if (file.size > MAX_FILE_SIZE) {
      showError(`${file.name}: Die Datei ist größer als 512 MiB.`);

      continue;
    }

    if (documents.length + result.length >= MAX_FILES) {
      showError(
        "Es können maximal 50 PDFs gleichzeitig zusammengefügt werden.",
      );

      break;
    }

    if (totalSize + file.size > MAX_TOTAL_SIZE) {
      showError("Die PDFs dürfen zusammen maximal 1 GiB groß sein.");

      break;
    }

    result.push(file);

    totalSize += file.size;
  }

  return result;
}

async function uploadPDF(file: File): Promise<PDFUploadResponse> {
  const body = new FormData();

  body.append("file", file, file.name);

  const response = await fetch("/pdf/uploads", {
    method: "POST",
    body,
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(message.trim() || "PDF konnte nicht hochgeladen werden.");
  }

  const data = (await response.json()) as unknown;

  if (!isPDFUploadResponse(data)) {
    throw new Error("Der Server hat eine ungültige Upload-Antwort geliefert.");
  }

  return data;
}

function isPDFUploadResponse(value: unknown): value is PDFUploadResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    typeof candidate.id === "string" &&
    typeof candidate.filename === "string" &&
    typeof candidate.size === "number" &&
    typeof candidate.page_count === "number" &&
    Array.isArray(candidate.previews) &&
    candidate.previews.every((preview) => typeof preview === "string")
  );
}

function renderDocuments(): void {
  const list = getElement<HTMLElement>("#pdf-merge-list");

  const selection = getElement<HTMLElement>("#pdf-merge-selection");

  const count = getElement<HTMLElement>("#pdf-merge-count");

  const documentTemplate = getElement<HTMLTemplateElement>(
    "#pdf-merge-document-template",
  );

  const pageTemplate = getElement<HTMLTemplateElement>(
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
    const item = createDocumentElement(
      documentTemplate,
      pageTemplate,
      document,
      index,
    );

    list.append(item);
  });

  updateControls();
}

function createDocumentElement(
  documentTemplate: HTMLTemplateElement,
  pageTemplate: HTMLTemplateElement,
  document: MergeDocument,
  index: number,
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
      moveDocument(document.id, -1);
    });
  }

  if (moveDown) {
    moveDown.disabled = index === documents.length - 1;

    moveDown.addEventListener("click", () => {
      moveDocument(document.id, 1);
    });
  }

  remove?.addEventListener("click", () => {
    void removeDocument(document.id);
  });

  setupDocumentDrag(item, document.id);

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

function setupDocumentDrag(item: HTMLElement, id: string): void {
  item.addEventListener("dragstart", (event: DragEvent) => {
    if (
      event.target instanceof Element &&
      event.target.closest(".pdf-page-strip")
    ) {
      event.preventDefault();

      return;
    }

    item.classList.add("dragging");

    event.dataTransfer?.setData("text/plain", id);

    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
    }
  });

  item.addEventListener("dragend", () => {
    item.classList.remove("dragging");

    clearDragTargets();
  });

  item.addEventListener("dragover", (event: DragEvent) => {
    event.preventDefault();

    const draggedID = event.dataTransfer?.getData("text/plain");

    if (draggedID && draggedID !== id) {
      clearDragTargets();

      item.classList.add("drag-target");
    }
  });

  item.addEventListener("dragleave", () => {
    item.classList.remove("drag-target");
  });

  item.addEventListener("drop", (event: DragEvent) => {
    event.preventDefault();

    const draggedID = event.dataTransfer?.getData("text/plain");

    item.classList.remove("drag-target");

    if (!draggedID || draggedID === id) {
      return;
    }

    reorderDocument(draggedID, id);
  });
}

function clearDragTargets(): void {
  root
    ?.querySelectorAll<HTMLElement>(".pdf-file-item.drag-target")
    .forEach((item) => {
      item.classList.remove("drag-target");
    });
}

function moveDocument(id: string, offset: number): void {
  const index = documents.findIndex((document) => document.id === id);

  if (index < 0) {
    return;
  }

  const destination = index + offset;

  if (destination < 0 || destination >= documents.length) {
    return;
  }

  const [document] = documents.splice(index, 1);

  if (!document) {
    return;
  }

  documents.splice(destination, 0, document);

  renderDocuments();
}

function reorderDocument(sourceID: string, targetID: string): void {
  const sourceIndex = documents.findIndex(
    (document) => document.id === sourceID,
  );

  const targetIndex = documents.findIndex(
    (document) => document.id === targetID,
  );

  if (sourceIndex < 0 || targetIndex < 0) {
    return;
  }

  const [document] = documents.splice(sourceIndex, 1);

  if (!document) {
    return;
  }

  const insertionIndex =
    sourceIndex < targetIndex ? targetIndex - 1 : targetIndex;

  documents.splice(insertionIndex, 0, document);

  renderDocuments();
}

async function removeDocument(id: string): Promise<void> {
  const index = documents.findIndex((document) => document.id === id);

  if (index < 0) {
    return;
  }

  documents.splice(index, 1);

  renderDocuments();

  await deleteUpload(id);
}

async function clearDocuments(): Promise<void> {
  if (uploading || merging) {
    return;
  }

  const ids = documents.map((document) => document.id);

  documents = [];

  renderDocuments();

  await Promise.allSettled(ids.map(deleteUpload));
}

async function deleteUpload(id: string): Promise<void> {
  try {
    const response = await fetch(`/pdf/uploads/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });

    if (!response.ok && response.status !== 404) {
      console.warn(`PDF upload ${id} could not be deleted.`);
    }
  } catch (error: unknown) {
    console.warn("PDF upload cleanup failed:", error);
  }
}

async function mergeDocuments(): Promise<void> {
  if (documents.length < 2 || uploading || merging) {
    return;
  }

  clearError();

  merging = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-merge-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const response = await fetch("/pdf/merge", {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        ids: documents.map((document) => document.id),
      }),
    });

    if (!response.ok) {
      const message = await response.text();

      throw new Error(
        message.trim() || "PDFs konnten nicht zusammengefügt werden.",
      );
    }

    const blob = await response.blob();

    downloadBlob(
      blob,
      getDownloadFilename(
        response.headers.get("Content-Disposition"),
        "zusammengefuegt.pdf",
      ),
    );

    /*
     * Das Backend löscht die verwendeten Uploads
     * nach erfolgreichem Merge.
     */
    documents = [];

    renderDocuments();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "PDFs konnten nicht zusammengefügt werden.",
    );
  } finally {
    merging = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

function updateControls(): void {
  const submit = getElement<HTMLButtonElement>("#pdf-merge-submit");

  const clear = getElement<HTMLButtonElement>("#pdf-merge-clear");

  const input = getElement<HTMLInputElement>("#pdf-merge-files");

  if (submit) {
    submit.disabled = documents.length < 2 || uploading || merging;
  }

  if (clear) {
    clear.disabled = documents.length === 0 || uploading || merging;
  }

  if (input) {
    input.disabled = uploading || merging || documents.length >= MAX_FILES;
  }
}

function updateUploadProgress(
  filename: string,
  current: number,
  total: number,
): void {
  const text = getElement<HTMLElement>("#pdf-upload-progress-text");

  if (!text) {
    return;
  }

  text.textContent = `${current} von ${total}: ${filename}`;
}

function getTotalSize(): number {
  return documents.reduce((total, document) => total + document.size, 0);
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

function showError(message: string): void {
  const error = getElement<HTMLElement>("#pdf-merge-error");

  if (!error) {
    return;
  }

  error.textContent = message;

  error.hidden = false;
}

function clearError(): void {
  const error = getElement<HTMLElement>("#pdf-merge-error");

  if (!error) {
    return;
  }

  error.textContent = "";

  error.hidden = true;
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
