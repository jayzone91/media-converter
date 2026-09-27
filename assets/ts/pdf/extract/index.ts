import { downloadBlob } from "../../shared/download.ts";

import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import {
  createPDFSelectionGrid,
  type PDFSelectionGrid,
} from "../shared/selection-grid.ts";

import { extractPDFPages } from "./api.ts";

const MAX_FILE_SIZE = 512 * 1024 * 1024;

let root: HTMLElement | null = null;

let upload: PDFUpload | null = null;

let grid: PDFSelectionGrid | null = null;

let selectedPages: number[] = [];

let uploading = false;

let processing = false;

export function setupPDFExtract(workspace: HTMLElement): void {
  grid?.destroy();

  root = workspace;

  upload = null;

  selectedPages = [];

  uploading = false;
  processing = false;

  grid = createPDFSelectionGrid({
    root: workspace,

    viewportSelector: "#pdf-extract-pages",

    templateSelector: "#pdf-select-page-template",

    onSelectionChange: handleSelectionChange,
  });

  const input = getElement<HTMLInputElement>("#pdf-extract-file");

  const dropZone = getElement<HTMLElement>("#pdf-extract-drop-zone");

  const reset = getElement<HTMLButtonElement>("#pdf-extract-reset-file");

  const selectAll = getElement<HTMLButtonElement>("#pdf-extract-select-all");

  const clearSelection = getElement<HTMLButtonElement>(
    "#pdf-extract-clear-selection",
  );

  const submit = getElement<HTMLButtonElement>("#pdf-extract-submit");

  if (
    !input ||
    !dropZone ||
    !reset ||
    !selectAll ||
    !clearSelection ||
    !submit
  ) {
    grid.destroy();

    grid = null;

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

  selectAll.addEventListener("click", () => {
    grid?.selectAll();
  });

  clearSelection.addEventListener("click", () => {
    grid?.clearSelection();
  });

  submit.addEventListener("click", () => {
    void createPDF();
  });

  render();
}

export async function destroyPDFExtract(): Promise<void> {
  const uploadID = upload?.id;

  grid?.destroy();

  grid = null;
  root = null;
  upload = null;

  selectedPages = [];

  uploading = false;
  processing = false;

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
      showError("Es kann nur eine PDF gleichzeitig bearbeitet werden.");

      return;
    }

    const file = files[0];

    if (file) {
      void selectFile(file);
    }
  });
}

async function selectFile(file: File): Promise<void> {
  if (uploading || processing) {
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
  }

  selectedPages = [];

  grid?.setPages([]);

  uploading = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-extract-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    upload = await uploadPDF(file);

    grid?.setPages(upload.previews);

    render();
  } catch (error: unknown) {
    upload = null;

    selectedPages = [];

    grid?.setPages([]);

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
  if (uploading || processing) {
    return;
  }

  const uploadID = upload?.id;

  upload = null;

  selectedPages = [];

  grid?.setPages([]);

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

function handleSelectionChange(pages: number[]): void {
  selectedPages = pages;

  updateSelectionText();

  updateControls();
}

async function createPDF(): Promise<void> {
  if (!upload || selectedPages.length === 0 || uploading || processing) {
    return;
  }

  clearError();

  processing = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-extract-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await extractPDFPages(upload.id, selectedPages);

    downloadBlob(result.blob, result.filename);

    upload = null;

    selectedPages = [];

    grid?.setPages([]);

    render();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die ausgewählten Seiten konnten nicht extrahiert werden.",
    );
  } finally {
    processing = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

function render(): void {
  const editor = getElement<HTMLElement>("#pdf-extract-editor");

  const dropZone = getElement<HTMLElement>("#pdf-extract-drop-zone");

  const filename = getElement<HTMLElement>("#pdf-extract-filename");

  const meta = getElement<HTMLElement>("#pdf-extract-meta");

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

  updateSelectionText();

  updateControls();
}

function updateSelectionText(): void {
  const element = getElement<HTMLElement>("#pdf-extract-selected-count");

  if (!element) {
    return;
  }

  if (selectedPages.length === 0) {
    element.textContent = "0 Seiten ausgewählt";

    return;
  }

  if (selectedPages.length === 1) {
    element.textContent = "1 Seite ausgewählt";

    return;
  }

  element.textContent = `${selectedPages.length} Seiten ausgewählt`;
}

function updateControls(): void {
  const input = getElement<HTMLInputElement>("#pdf-extract-file");

  const reset = getElement<HTMLButtonElement>("#pdf-extract-reset-file");

  const selectAll = getElement<HTMLButtonElement>("#pdf-extract-select-all");

  const clearSelection = getElement<HTMLButtonElement>(
    "#pdf-extract-clear-selection",
  );

  const submit = getElement<HTMLButtonElement>("#pdf-extract-submit");

  const busy = uploading || processing;

  if (input) {
    input.disabled = busy;
  }

  if (reset) {
    reset.disabled = !upload || busy;
  }

  if (selectAll) {
    selectAll.disabled =
      !upload || busy || selectedPages.length === upload.pageCount;
  }

  if (clearSelection) {
    clearSelection.disabled = selectedPages.length === 0 || busy;
  }

  if (submit) {
    submit.disabled = !upload || selectedPages.length === 0 || busy;
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-extract-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-extract-error");

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
