import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import { clearEditorState, editorState } from "./state.ts";

import {
  renderImageObjects,
  setupPDFEditImage,
  updateImageProperties,
} from "./image.ts";

import {
  renderDrawStrokes,
  resetPDFEditDraw,
  setupPDFEditDraw,
} from "./draw.ts";

import { setupPDFEditSubmit, updatePDFEditSubmitState } from "./submit.ts";

import {
  renderTextObjects,
  setupPDFEditText,
  updatePageChangeIndicators,
  updateTextProperties,
} from "./text.ts";

let previewResizeObserver: ResizeObserver | null = null;

export function setupPDFEdit(workspace: HTMLElement): void {
  editorState.root = workspace;

  const input = workspace.querySelector<HTMLInputElement>("#pdf-edit-file");

  const dropZone = workspace.querySelector<HTMLElement>("#pdf-edit-drop-zone");

  const resetButton = workspace.querySelector<HTMLButtonElement>(
    "#pdf-edit-reset-file",
  );

  const preview = workspace.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

  if (!input || !dropZone || !resetButton || !preview) {
    return;
  }

  if (
    !setupPDFEditText() ||
    !setupPDFEditImage() ||
    !setupPDFEditDraw() ||
    !setupPDFEditSubmit()
  ) {
    return;
  }

  setupPreviewScaling(preview);

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
  previewResizeObserver?.disconnect();

  previewResizeObserver = null;

  resetPDFEditDraw();

  if (editorState.activeUpload) {
    await deletePDFUpload(editorState.activeUpload.id);
  }

  clearEditorState();

  editorState.activeUpload = null;

  editorState.activePage = 0;

  editorState.root = null;
}

function setupPreviewScaling(preview: HTMLImageElement): void {
  previewResizeObserver?.disconnect();

  previewResizeObserver = new ResizeObserver(() => {
    renderTextObjects();
    renderImageObjects();
    renderDrawStrokes();
  });

  previewResizeObserver.observe(preview);

  preview.addEventListener("load", () => {
    renderTextObjects();
    renderImageObjects();
    renderDrawStrokes();
  });
}

async function selectFile(file: File): Promise<void> {
  if (!editorState.root) {
    return;
  }

  hideError();

  if (!isPDFFile(file)) {
    showError("Bitte eine PDF-Datei auswählen.");

    return;
  }

  setUploading(true);

  try {
    if (editorState.activeUpload) {
      await deletePDFUpload(editorState.activeUpload.id);

      editorState.activeUpload = null;
    }

    resetPDFEditDraw();
    clearEditorState();

    editorState.activeUpload = await uploadPDF(file);

    editorState.activePage = 0;

    renderUpload(editorState.activeUpload);

    updatePDFEditSubmitState();
  } catch (error: unknown) {
    showError(errorMessage(error));
  } finally {
    setUploading(false);
  }
}

function renderUpload(upload: PDFUpload): void {
  const root = editorState.root;

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
  const root = editorState.root;

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

  updatePageChangeIndicators();
}

function selectPage(index: number): void {
  const root = editorState.root;

  const upload = editorState.activeUpload;

  if (!root || !upload) {
    return;
  }

  if (index < 0 || index >= upload.pageCount) {
    return;
  }

  editorState.activePage = index;

  editorState.selectedTextID = null;

  editorState.selectedImageID = null;

  editorState.dragState = null;

  const preview = root.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

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

  renderTextObjects();
  renderImageObjects();
  renderDrawStrokes();

  updateTextProperties();
  updateImageProperties();
}

async function resetEditor(): Promise<void> {
  const root = editorState.root;

  if (!root) {
    return;
  }

  if (editorState.activeUpload) {
    await deletePDFUpload(editorState.activeUpload.id);

    editorState.activeUpload = null;
  }

  resetPDFEditDraw();
  clearEditorState();

  editorState.activePage = 0;

  const input = root.querySelector<HTMLInputElement>("#pdf-edit-file");

  const dropZone = root.querySelector<HTMLElement>("#pdf-edit-drop-zone");

  const editor = root.querySelector<HTMLElement>("#pdf-edit-editor");

  const pages = root.querySelector<HTMLElement>("#pdf-edit-pages");

  const preview = root.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

  const textOverlay = root.querySelector<HTMLElement>("#pdf-edit-overlay");

  const imageOverlay = root.querySelector<HTMLElement>(
    "#pdf-edit-image-overlay",
  );

  const drawOverlay = root.querySelector<SVGSVGElement>(
    "#pdf-edit-draw-overlay",
  );

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

  textOverlay?.replaceChildren();
  imageOverlay?.replaceChildren();
  drawOverlay?.replaceChildren();

  if (preview) {
    preview.removeAttribute("src");

    preview.alt = "";
  }

  hideError();

  updatePDFEditSubmitState();
}

function setUploading(uploading: boolean): void {
  const root = editorState.root;

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
  const element =
    editorState.root?.querySelector<HTMLElement>("#pdf-edit-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function hideError(): void {
  const element =
    editorState.root?.querySelector<HTMLElement>("#pdf-edit-error");

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
