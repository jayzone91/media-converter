import { downloadBlob } from "../../shared/download.ts";

import { deletePDFUpload } from "../uploads.ts";

import { encryptPDF } from "../security/api.ts";

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

export function setupPDFEncrypt(workspace: HTMLElement): void {
  root = workspace;

  upload = null;

  busy = false;

  const input = getElement<HTMLInputElement>("#pdf-encrypt-file");

  const dropZone = getElement<HTMLElement>("#pdf-encrypt-drop-zone");

  const reset = getElement<HTMLButtonElement>("#pdf-encrypt-reset-file");

  const password = getElement<HTMLInputElement>("#pdf-encrypt-password");

  const confirm = getElement<HTMLInputElement>("#pdf-encrypt-password-confirm");

  const showPassword = getElement<HTMLInputElement>(
    "#pdf-encrypt-show-password",
  );

  const submit = getElement<HTMLButtonElement>("#pdf-encrypt-submit");

  if (
    !input ||
    !dropZone ||
    !reset ||
    !password ||
    !confirm ||
    !showPassword ||
    !submit
  ) {
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

  password.addEventListener("input", updateControls);

  confirm.addEventListener("input", updateControls);

  showPassword.addEventListener("change", () => {
    const type = showPassword.checked ? "text" : "password";

    password.type = type;

    confirm.type = type;
  });

  reset.addEventListener("click", () => {
    void resetUpload();
  });

  submit.addEventListener("click", () => {
    void encryptCurrentPDF();
  });

  render();
}

export async function destroyPDFEncrypt(): Promise<void> {
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

  clearPasswords();

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

async function encryptCurrentPDF(): Promise<void> {
  const password = getElement<HTMLInputElement>("#pdf-encrypt-password");

  const confirm = getElement<HTMLInputElement>("#pdf-encrypt-password-confirm");

  if (!upload || !password || !confirm || busy) {
    return;
  }

  if (password.value === "") {
    showError("Bitte ein Passwort eingeben.");

    return;
  }

  if (password.value !== confirm.value) {
    showError("Die Passwörter stimmen nicht überein.");

    return;
  }

  clearError();

  busy = true;

  render();

  const progress = getElement<HTMLElement>("#pdf-encrypt-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await encryptPDF(upload.id, password.value);

    downloadBlob(result.blob, result.filename);

    upload = null;

    clearPasswords();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die PDF konnte nicht verschlüsselt werden.",
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
  const editor = getElement<HTMLElement>("#pdf-encrypt-editor");

  const dropZone = getElement<HTMLElement>("#pdf-encrypt-drop-zone");

  const filename = getElement<HTMLElement>("#pdf-encrypt-filename");

  const meta = getElement<HTMLElement>("#pdf-encrypt-meta");

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
  const password = getElement<HTMLInputElement>("#pdf-encrypt-password");

  const confirm = getElement<HTMLInputElement>("#pdf-encrypt-password-confirm");

  const reset = getElement<HTMLButtonElement>("#pdf-encrypt-reset-file");

  const submit = getElement<HTMLButtonElement>("#pdf-encrypt-submit");

  if (reset) {
    reset.disabled = !upload || busy;
  }

  if (password) {
    password.disabled = busy;
  }

  if (confirm) {
    confirm.disabled = busy;
  }

  if (submit) {
    submit.disabled =
      !upload ||
      !password ||
      !confirm ||
      password.value === "" ||
      password.value !== confirm.value ||
      busy;
  }
}

function clearPasswords(): void {
  const password = getElement<HTMLInputElement>("#pdf-encrypt-password");

  const confirm = getElement<HTMLInputElement>("#pdf-encrypt-password-confirm");

  if (password) {
    password.value = "";
  }

  if (confirm) {
    confirm.value = "";
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-encrypt-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-encrypt-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
