import { downloadURL } from "../../shared/download.ts";

import { deletePDFUpload } from "../uploads.ts";

import { decryptPDF } from "../security/api.ts";

import {
  formatPDFSecuritySize,
  setupPDFSecurityDropZone,
  validatePDFSecurityFile,
} from "../security/helpers.ts";

import {
  type PDFSecurityUpload,
  uploadPDFSecurityFile,
} from "../security/upload.ts";

let root: HTMLElement | null = null;

let upload: PDFSecurityUpload | null = null;

let busy = false;

export function setupPDFDecrypt(workspace: HTMLElement): void {
  root = workspace;

  upload = null;

  busy = false;

  const input = getElement<HTMLInputElement>("#pdf-decrypt-file");

  const dropZone = getElement<HTMLElement>("#pdf-decrypt-drop-zone");

  const reset = getElement<HTMLButtonElement>("#pdf-decrypt-reset-file");

  const password = getElement<HTMLInputElement>("#pdf-decrypt-password");

  const showPassword = getElement<HTMLInputElement>(
    "#pdf-decrypt-show-password",
  );

  const submit = getElement<HTMLButtonElement>("#pdf-decrypt-submit");

  if (!input || !dropZone || !reset || !password || !showPassword || !submit) {
    return;
  }

  input.addEventListener("change", () => {
    const file = input.files?.[0];

    input.value = "";

    if (file) {
      void selectFile(file);
    }
  });

  setupPDFSecurityDropZone(
    dropZone,
    (file) => {
      void selectFile(file);
    },
    showError,
  );

  showPassword.addEventListener("change", () => {
    password.type = showPassword.checked ? "text" : "password";
  });

  reset.addEventListener("click", () => {
    void resetUpload();
  });

  submit.addEventListener("click", () => {
    void decryptCurrentPDF();
  });

  render();
}

export async function destroyPDFDecrypt(): Promise<void> {
  const uploadID = upload?.id;

  root = null;

  upload = null;

  busy = false;

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

async function selectFile(file: File): Promise<void> {
  if (busy) {
    return;
  }

  clearError();

  const validation = validatePDFSecurityFile(file);

  if (validation) {
    showError(validation);

    return;
  }

  if (upload) {
    await deletePDFUpload(upload.id);

    upload = null;
  }

  busy = true;

  render();

  try {
    upload = await uploadPDFSecurityFile(file);
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "PDF konnte nicht hochgeladen werden.",
    );
  } finally {
    busy = false;

    render();
  }
}

async function resetUpload(): Promise<void> {
  if (busy) {
    return;
  }

  const uploadID = upload?.id;

  upload = null;

  clearPassword();

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

async function decryptCurrentPDF(): Promise<void> {
  const password = getElement<HTMLInputElement>("#pdf-decrypt-password");

  if (!upload || !password || busy) {
    return;
  }

  clearError();

  busy = true;

  render();

  const progress = getElement<HTMLElement>("#pdf-decrypt-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await decryptPDF(upload.id, password.value);

    downloadURL(result.downloadURL);

    upload = null;

    clearPassword();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Das Passwort konnte nicht entfernt werden.",
    );
  } finally {
    busy = false;

    if (progress) {
      progress.hidden = true;
    }

    render();
  }
}

function render(): void {
  const editor = getElement<HTMLElement>("#pdf-decrypt-editor");

  const dropZone = getElement<HTMLElement>("#pdf-decrypt-drop-zone");

  const filename = getElement<HTMLElement>("#pdf-decrypt-filename");

  const meta = getElement<HTMLElement>("#pdf-decrypt-meta");

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
    meta.textContent = formatPDFSecuritySize(upload.size);
  }

  updateControls();
}

function updateControls(): void {
  const password = getElement<HTMLInputElement>("#pdf-decrypt-password");

  const reset = getElement<HTMLButtonElement>("#pdf-decrypt-reset-file");

  const submit = getElement<HTMLButtonElement>("#pdf-decrypt-submit");

  if (password) {
    password.disabled = busy;
  }

  if (reset) {
    reset.disabled = !upload || busy;
  }

  if (submit) {
    submit.disabled = !upload || busy;
  }
}

function clearPassword(): void {
  const password = getElement<HTMLInputElement>("#pdf-decrypt-password");

  if (password) {
    password.value = "";
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-decrypt-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-decrypt-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
