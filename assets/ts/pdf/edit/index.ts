import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

let activeUpload: PDFUpload | null = null;

let activePage = 0;

let root: HTMLElement | null = null;

export function setupPDFEdit(workspace: HTMLElement): void {
  root = workspace;

  const input = workspace.querySelector<HTMLInputElement>("#pdf-edit-file");

  const dropZone = workspace.querySelector<HTMLElement>("#pdf-edit-drop-zone");

  const resetButton = workspace.querySelector<HTMLButtonElement>(
    "#pdf-edit-reset-file",
  );

  if (!input || !dropZone || !resetButton) {
    return;
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
    void resetEditor();
  });
}

export async function destroyPDFEdit(): Promise<void> {
  if (activeUpload) {
    await deletePDFUpload(activeUpload.id);
  }

  activeUpload = null;
  activePage = 0;
  root = null;
}

async function selectFile(file: File): Promise<void> {
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
    if (activeUpload) {
      await deletePDFUpload(activeUpload.id);

      activeUpload = null;
    }

    activeUpload = await uploadPDF(file);

    activePage = 0;

    renderUpload(activeUpload);
  } catch (error: unknown) {
    showError(errorMessage(error));
  } finally {
    setUploading(false);
  }
}

function renderUpload(upload: PDFUpload): void {
  if (!root) {
    return;
  }

  const dropZone = root.querySelector<HTMLElement>("#pdf-edit-drop-zone");

  const editor = root.querySelector<HTMLElement>("#pdf-edit-editor");

  const filename = root.querySelector<HTMLElement>("#pdf-edit-filename");

  const meta = root.querySelector<HTMLElement>("#pdf-edit-meta");

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
  if (!root) {
    return;
  }

  const container = root.querySelector<HTMLElement>("#pdf-edit-pages");

  const template = root.querySelector<HTMLTemplateElement>(
    "#pdf-edit-page-template",
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
}

function selectPage(index: number): void {
  if (!root || !activeUpload) {
    return;
  }

  if (index < 0 || index >= activeUpload.pageCount) {
    return;
  }

  activePage = index;

  const preview = root.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

  if (!preview) {
    return;
  }

  preview.src = activeUpload.previews[index] ?? "";
  preview.alt = `PDF-Seite ${index + 1}`;

  root
    .querySelectorAll<HTMLButtonElement>(".pdf-edit-page")
    .forEach((button) => {
      const page = Number(button.dataset.page);

      const selected = page === index;

      button.classList.toggle("selected", selected);

      button.setAttribute("aria-current", selected ? "page" : "false");
    });
}

async function resetEditor(): Promise<void> {
  if (!root) {
    return;
  }

  if (activeUpload) {
    await deletePDFUpload(activeUpload.id);

    activeUpload = null;
  }

  activePage = 0;

  const input = root.querySelector<HTMLInputElement>("#pdf-edit-file");

  const dropZone = root.querySelector<HTMLElement>("#pdf-edit-drop-zone");

  const editor = root.querySelector<HTMLElement>("#pdf-edit-editor");

  const pages = root.querySelector<HTMLElement>("#pdf-edit-pages");

  const preview = root.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

  const overlay = root.querySelector<HTMLElement>("#pdf-edit-overlay");

  if (input) {
    input.value = "";
  }

  if (dropZone) {
    dropZone.hidden = false;
  }

  if (editor) {
    editor.hidden = true;
  }

  pages?.replaceChildren();
  overlay?.replaceChildren();

  if (preview) {
    preview.removeAttribute("src");
    preview.alt = "";
  }

  hideError();
}

function setUploading(uploading: boolean): void {
  if (!root) {
    return;
  }

  const progress = root.querySelector<HTMLElement>("#pdf-edit-upload-progress");

  const dropZone = root.querySelector<HTMLElement>("#pdf-edit-drop-zone");

  if (progress) {
    progress.hidden = !uploading;
  }

  if (dropZone) {
    dropZone.classList.toggle("busy", uploading);
  }
}

function showError(message: string): void {
  if (!root) {
    return;
  }

  const element = root.querySelector<HTMLElement>("#pdf-edit-error");

  if (!element) {
    return;
  }

  element.textContent = message;
  element.hidden = false;
}

function hideError(): void {
  if (!root) {
    return;
  }

  const element = root.querySelector<HTMLElement>("#pdf-edit-error");

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

  const decimals = unit === 0 ? 0 : 1;

  return `${value.toFixed(decimals)} ${units[unit]}`;
}

function pageLabel(count: number): string {
  if (count === 1) {
    return "1 Seite";
  }

  return `${count} Seiten`;
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return "PDF konnte nicht geladen werden.";
}
