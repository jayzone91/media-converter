import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import { clearRedactState, redactState } from "./state.ts";

import {
  renderRedactions,
  updateControls,
  updatePageIndicators,
} from "./render.ts";

import { resetRedactionInteraction } from "./interaction.ts";

export function setupRedactUpload(): boolean {
  const root = redactState.root;

  if (!root) {
    return false;
  }

  const input = root.querySelector<HTMLInputElement>("#pdf-redact-file");

  const dropZone = root.querySelector<HTMLElement>("#pdf-redact-drop-zone");

  const resetButton = root.querySelector<HTMLButtonElement>(
    "#pdf-redact-reset-file",
  );

  if (!input || !dropZone || !resetButton) {
    return false;
  }

  input.addEventListener("change", () => {
    const file = input.files?.[0];

    if (!file) {
      return;
    }

    void selectFile(file);
  });

  dropZone.addEventListener("dragover", (event) => {
    event.preventDefault();

    dropZone.classList.add("drag-over");
  });

  dropZone.addEventListener("dragleave", () => {
    dropZone.classList.remove("drag-over");
  });

  dropZone.addEventListener("drop", (event) => {
    event.preventDefault();

    dropZone.classList.remove("drag-over");

    const file = event.dataTransfer?.files[0];

    if (!file) {
      return;
    }

    void selectFile(file);
  });

  resetButton.addEventListener("click", () => {
    void resetWorkspace();
  });

  return true;
}

export async function destroyRedactUpload(): Promise<void> {
  if (redactState.activeUpload) {
    await deletePDFUpload(redactState.activeUpload.id);
  }

  clearRedactState();

  redactState.activeUpload = null;

  redactState.activePage = 0;

  resetRedactionInteraction();
}

async function selectFile(file: File): Promise<void> {
  const root = redactState.root;

  if (!root) {
    return;
  }

  hideError();

  if (!isPDFFile(file)) {
    showError("Bitte eine PDF-Datei auswählen.");

    return;
  }

  setUploading(true);

  try {
    if (redactState.activeUpload) {
      await deletePDFUpload(redactState.activeUpload.id);
    }

    clearRedactState();

    redactState.activeUpload = await uploadPDF(file);

    redactState.activePage = 0;

    renderUpload(redactState.activeUpload);
  } catch (error: unknown) {
    showError(errorMessage(error));
  } finally {
    setUploading(false);
  }
}

function renderUpload(upload: PDFUpload): void {
  const root = redactState.root;

  if (!root) {
    return;
  }

  const dropZone = root.querySelector<HTMLElement>("#pdf-redact-drop-zone");

  const editor = root.querySelector<HTMLElement>("#pdf-redact-editor");

  const filename = root.querySelector<HTMLElement>("#pdf-redact-filename");

  const meta = root.querySelector<HTMLElement>("#pdf-redact-meta");

  if (!dropZone || !editor || !filename || !meta) {
    return;
  }

  dropZone.hidden = true;

  editor.hidden = false;

  filename.textContent = upload.filename;

  meta.textContent = `${formatFileSize(upload.size)} · ${pageLabel(
    upload.pageCount,
  )}`;

  renderPages(upload);

  selectPage(0);
}

function renderPages(upload: PDFUpload): void {
  const root = redactState.root;

  if (!root) {
    return;
  }

  const container = root.querySelector<HTMLElement>("#pdf-redact-pages");

  const template = root.querySelector<HTMLTemplateElement>(
    "#pdf-redact-page-template",
  );

  if (!container || !template) {
    return;
  }

  container.replaceChildren();

  upload.previews.forEach((preview, index) => {
    const fragment = template.content.cloneNode(true);

    if (!(fragment instanceof DocumentFragment)) {
      return;
    }

    const button = fragment.querySelector<HTMLButtonElement>(".pdf-edit-page");

    const number = fragment.querySelector<HTMLElement>(".pdf-edit-page-number");

    const image = fragment.querySelector<HTMLImageElement>("img");

    if (!button || !number || !image) {
      return;
    }

    button.dataset.page = String(index);

    button.setAttribute("aria-label", `Seite ${index + 1} auswählen`);

    number.textContent = `Seite ${index + 1}`;

    image.src = preview;

    image.alt = `Vorschau Seite ${index + 1}`;

    button.addEventListener("click", () => {
      selectPage(index);
    });

    container.append(fragment);
  });

  updatePageIndicators();
}

function selectPage(index: number): void {
  const root = redactState.root;

  const upload = redactState.activeUpload;

  if (!root || !upload) {
    return;
  }

  if (index < 0 || index >= upload.pageCount) {
    return;
  }

  redactState.activePage = index;

  redactState.selectedID = null;

  resetRedactionInteraction();

  const preview = root.querySelector<HTMLImageElement>("#pdf-redact-preview");

  if (!preview) {
    return;
  }

  preview.src = upload.previews[index] ?? "";

  preview.alt = `PDF-Seite ${index + 1}`;

  root
    .querySelectorAll<HTMLButtonElement>(".pdf-edit-page")
    .forEach((button) => {
      const page = Number(button.dataset.page);

      const selected = page === index;

      button.classList.toggle("selected", selected);

      button.setAttribute("aria-current", selected ? "page" : "false");
    });

  renderRedactions();
  updateControls();
}

async function resetWorkspace(): Promise<void> {
  const root = redactState.root;

  if (!root) {
    return;
  }

  if (redactState.activeUpload) {
    await deletePDFUpload(redactState.activeUpload.id);
  }

  clearRedactState();

  redactState.activeUpload = null;

  redactState.activePage = 0;

  resetRedactionInteraction();

  const input = root.querySelector<HTMLInputElement>("#pdf-redact-file");

  const dropZone = root.querySelector<HTMLElement>("#pdf-redact-drop-zone");

  const editor = root.querySelector<HTMLElement>("#pdf-redact-editor");

  const pages = root.querySelector<HTMLElement>("#pdf-redact-pages");

  const preview = root.querySelector<HTMLImageElement>("#pdf-redact-preview");

  root.querySelector<HTMLElement>("#pdf-redact-overlay")?.replaceChildren();

  pages?.replaceChildren();

  if (input) {
    input.value = "";
  }

  if (dropZone) {
    dropZone.hidden = false;
  }

  if (editor) {
    editor.hidden = true;
  }

  if (preview) {
    preview.removeAttribute("src");

    preview.alt = "";
  }

  hideError();

  updateControls();
}

function setUploading(uploading: boolean): void {
  const root = redactState.root;

  if (!root) {
    return;
  }

  const progress = root.querySelector<HTMLElement>(
    "#pdf-redact-upload-progress",
  );

  const dropZone = root.querySelector<HTMLElement>("#pdf-redact-drop-zone");

  if (progress) {
    progress.hidden = !uploading;
  }

  if (dropZone) {
    dropZone.classList.toggle("busy", uploading);
  }
}

function showError(message: string): void {
  const element =
    redactState.root?.querySelector<HTMLElement>("#pdf-redact-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function hideError(): void {
  const element =
    redactState.root?.querySelector<HTMLElement>("#pdf-redact-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function formatFileSize(bytes: number): string {
  const units = ["B", "KiB", "MiB", "GiB"];

  let value = bytes;
  let unit = 0;

  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;

    unit++;
  }

  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function pageLabel(count: number): string {
  return count === 1 ? "1 Seite" : `${count} Seiten`;
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return "PDF konnte nicht geladen werden.";
}
