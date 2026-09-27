import { downloadBlob } from "../../shared/download.ts";
import { deletePDFUpload, mergePDFUploads, uploadPDF } from "./api.ts";
import { renderMergeDocuments } from "./render.ts";
import type { DocumentRenderActions, MergeDocument } from "./types.ts";

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

  render();
}

export async function destroyPDFMerge(): Promise<void> {
  const ids = documents.map((document) => document.id);

  documents = [];
  root = null;
  uploading = false;
  merging = false;

  await Promise.allSettled(ids.map(deletePDFUpload));
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

        render();
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

function render(): void {
  if (!root) {
    return;
  }

  const actions: DocumentRenderActions = {
    moveUp: (id) => {
      moveDocument(id, -1);
    },

    moveDown: (id) => {
      moveDocument(id, 1);
    },

    remove: (id) => {
      void removeDocument(id);
    },

    reorder: reorderDocument,
  };

  renderMergeDocuments(root, documents, actions);

  updateControls();
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

  const document = documents[index];

  if (!document) {
    return;
  }

  documents.splice(index, 1);

  documents.splice(destination, 0, document);

  render();
}

function reorderDocument(sourceID: string, targetID: string): void {
  const sourceIndex = documents.findIndex(
    (document) => document.id === sourceID,
  );

  const targetIndex = documents.findIndex(
    (document) => document.id === targetID,
  );

  if (sourceIndex < 0 || targetIndex < 0 || sourceIndex === targetIndex) {
    return;
  }

  const document = documents[sourceIndex];

  if (!document) {
    return;
  }

  documents.splice(sourceIndex, 1);

  const destination = sourceIndex < targetIndex ? targetIndex - 1 : targetIndex;

  documents.splice(destination, 0, document);

  render();
}

async function removeDocument(id: string): Promise<void> {
  const index = documents.findIndex((document) => document.id === id);

  if (index < 0) {
    return;
  }

  documents.splice(index, 1);

  render();

  await deletePDFUpload(id);
}

async function clearDocuments(): Promise<void> {
  if (uploading || merging) {
    return;
  }

  const ids = documents.map((document) => document.id);

  documents = [];

  render();

  await Promise.allSettled(ids.map(deletePDFUpload));
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
    const result = await mergePDFUploads(
      documents.map((document) => document.id),
    );

    downloadBlob(result.blob, result.filename);

    documents = [];

    render();
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
