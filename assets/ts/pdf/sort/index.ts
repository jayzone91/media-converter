import { downloadBlob } from "../../shared/download.ts";

import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import { sortPDFPages } from "./api.ts";

import { createSortPageRenderer, type SortPageRenderer } from "./render.ts";

import type { SortPage } from "./types.ts";

const MAX_FILE_SIZE = 512 * 1024 * 1024;

let root: HTMLElement | null = null;

let upload: PDFUpload | null = null;

let pages: SortPage[] = [];

let renderer: SortPageRenderer | null = null;

let uploading = false;
let sorting = false;

export function setupPDFSort(workspace: HTMLElement): void {
  renderer?.destroy();

  root = workspace;

  upload = null;

  pages = [];

  uploading = false;

  sorting = false;

  renderer = createSortPageRenderer(workspace, reorderPage);

  const input = getElement<HTMLInputElement>("#pdf-sort-file");

  const dropZone = getElement<HTMLElement>("#pdf-sort-drop-zone");

  const reset = getElement<HTMLButtonElement>("#pdf-sort-reset-file");

  const submit = getElement<HTMLButtonElement>("#pdf-sort-submit");

  if (!input || !dropZone || !reset || !submit) {
    renderer.destroy();

    renderer = null;

    return;
  }

  input.addEventListener("change", () => {
    const file = input.files?.[0];

    input.value = "";

    if (file) {
      void selectFile(file);
    }
  });

  setupDropZone(dropZone);

  reset.addEventListener("click", () => {
    void resetUpload();
  });

  submit.addEventListener("click", () => {
    void createSortedPDF();
  });

  render();
}

export async function destroyPDFSort(): Promise<void> {
  const uploadID = upload?.id;

  renderer?.destroy();

  renderer = null;

  root = null;

  upload = null;

  pages = [];

  uploading = false;

  sorting = false;

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
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

    if (files.length > 1) {
      showError(
        "Zum Sortieren kann nur eine PDF gleichzeitig ausgewählt werden.",
      );

      return;
    }

    const file = files[0];

    if (file) {
      void selectFile(file);
    }
  });
}

async function selectFile(file: File): Promise<void> {
  if (uploading || sorting) {
    return;
  }

  clearError();

  if (!isPDFFile(file)) {
    showError("Es können nur PDF-Dateien hochgeladen werden.");

    return;
  }

  if (file.size <= 0) {
    showError("Die PDF-Datei ist leer.");

    return;
  }

  if (file.size > MAX_FILE_SIZE) {
    showError("Die PDF-Datei ist größer als 512 MiB.");

    return;
  }

  if (upload) {
    await deletePDFUpload(upload.id);

    upload = null;

    pages = [];
  }

  uploading = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-sort-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    upload = await uploadPDF(file);

    pages = upload.previews.map((preview, index) => ({
      originalPage: index + 1,

      preview,
    }));

    render();

    const viewport = getElement<HTMLElement>("#pdf-sort-pages");

    if (viewport) {
      viewport.scrollTop = 0;
    }
  } catch (error: unknown) {
    upload = null;

    pages = [];

    renderer?.setPages([]);

    showError(
      error instanceof Error
        ? `${file.name}: ${error.message}`
        : `${file.name}: Upload fehlgeschlagen.`,
    );
  } finally {
    uploading = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

async function resetUpload(): Promise<void> {
  if (uploading || sorting) {
    return;
  }

  const uploadID = upload?.id;

  upload = null;

  pages = [];

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

function reorderPage(sourcePage: number, targetPage: number): void {
  const sourceIndex = pages.findIndex(
    (page) => page.originalPage === sourcePage,
  );

  const targetIndex = pages.findIndex(
    (page) => page.originalPage === targetPage,
  );

  if (sourceIndex < 0 || targetIndex < 0 || sourceIndex === targetIndex) {
    return;
  }

  const page = pages[sourceIndex];

  if (!page) {
    return;
  }

  pages.splice(sourceIndex, 1);

  /*
   * Wir interpretieren das Ziel als konkrete
   * Zielposition. Nach dem Entfernen der Quelle
   * muss bei einer Bewegung nach rechts/unten
   * der Index um eins korrigiert werden.
   */
  const destination = sourceIndex < targetIndex ? targetIndex - 1 : targetIndex;

  pages.splice(destination, 0, page);

  renderPages();

  updateControls();
}

async function createSortedPDF(): Promise<void> {
  if (
    !upload ||
    pages.length < 2 ||
    uploading ||
    sorting ||
    !hasChangedOrder()
  ) {
    return;
  }

  clearError();

  sorting = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-sort-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await sortPDFPages(
      upload.id,
      pages.map((page) => page.originalPage),
    );

    downloadBlob(result.blob, result.filename);

    /*
     * Das Backend entfernt den Upload
     * nach erfolgreicher Verarbeitung.
     */
    upload = null;

    pages = [];

    render();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "PDF-Seiten konnten nicht sortiert werden.",
    );
  } finally {
    sorting = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

function render(): void {
  const editor = getElement<HTMLElement>("#pdf-sort-editor");

  const dropZone = getElement<HTMLElement>("#pdf-sort-drop-zone");

  const filename = getElement<HTMLElement>("#pdf-sort-filename");

  const meta = getElement<HTMLElement>("#pdf-sort-meta");

  if (editor) {
    editor.hidden = upload === null;
  }

  if (dropZone) {
    dropZone.hidden = upload !== null;
  }

  if (upload && filename) {
    filename.textContent = upload.filename;
  }

  if (upload && meta) {
    meta.textContent = `${upload.pageCount} Seiten · ${formatBytes(upload.size)}`;
  }

  renderPages();

  updateControls();
}

function renderPages(): void {
  renderer?.setPages(pages);
}

function hasChangedOrder(): boolean {
  return pages.some((page, index) => page.originalPage !== index + 1);
}

function updateControls(): void {
  const input = getElement<HTMLInputElement>("#pdf-sort-file");

  const reset = getElement<HTMLButtonElement>("#pdf-sort-reset-file");

  const submit = getElement<HTMLButtonElement>("#pdf-sort-submit");

  if (input) {
    input.disabled = uploading || sorting;
  }

  if (reset) {
    reset.disabled = !upload || uploading || sorting;
  }

  if (submit) {
    submit.disabled =
      !upload || pages.length < 2 || !hasChangedOrder() || uploading || sorting;
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-sort-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-sort-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function formatBytes(bytes: number): string {
  const units = ["B", "KiB", "MiB", "GiB"] as const;

  if (bytes <= 0) {
    return "0 B";
  }

  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );

  const value = bytes / 1024 ** index;

  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index] ?? "B"}`;
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
