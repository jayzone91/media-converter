import { deletePDFUpload } from "../uploads.ts";

import { clearRedactState, redactState } from "./state.ts";

export interface PDFRedactResult {
  download_url: string;
  filename: string;

  preview_upload_id: string;
  previews: string[];
  page_count: number;
}

export function showRedactResult(result: PDFRedactResult): void {
  const root = redactState.root;

  if (!root) {
    return;
  }

  redactState.resultUploadID = result.preview_upload_id;

  redactState.resultDownloadURL = result.download_url;

  const editor = root.querySelector<HTMLElement>("#pdf-redact-editor");

  const resultSection = root.querySelector<HTMLElement>("#pdf-redact-result");

  const pages = root.querySelector<HTMLElement>("#pdf-redact-result-pages");

  const download = root.querySelector<HTMLButtonElement>(
    "#pdf-redact-result-download",
  );

  if (!resultSection || !pages || !download) {
    return;
  }

  if (editor) {
    editor.hidden = true;
  }

  resultSection.hidden = false;

  pages.replaceChildren();

  result.previews.forEach((preview, index) => {
    const link = document.createElement("a");

    link.className = "pdf-redact-result-page";

    link.href = preview;

    link.target = "_blank";

    link.rel = "noopener";

    link.setAttribute("aria-label", `Ergebnis Seite ${index + 1} öffnen`);

    const number = document.createElement("span");

    number.textContent = `Seite ${index + 1}`;

    const image = document.createElement("img");

    image.src = preview;

    image.alt = `Geschwärzte PDF – Seite ${index + 1}`;

    image.loading = "lazy";

    image.decoding = "async";

    link.append(number, image);

    pages.append(link);
  });

  download.disabled = false;

  download.onclick = () => {
    if (!redactState.resultDownloadURL) {
      return;
    }

    window.location.assign(redactState.resultDownloadURL);

    redactState.resultDownloadURL = null;

    download.disabled = true;
  };
}

export async function clearRedactResult(): Promise<void> {
  if (redactState.resultUploadID) {
    await deletePDFUpload(redactState.resultUploadID);
  }

  redactState.resultUploadID = null;

  redactState.resultDownloadURL = null;

  const root = redactState.root;

  if (!root) {
    return;
  }

  const resultSection = root.querySelector<HTMLElement>("#pdf-redact-result");

  const pages = root.querySelector<HTMLElement>("#pdf-redact-result-pages");

  const download = root.querySelector<HTMLButtonElement>(
    "#pdf-redact-result-download",
  );

  if (resultSection) {
    resultSection.hidden = true;
  }

  pages?.replaceChildren();

  if (download) {
    download.disabled = true;

    download.onclick = null;
  }

  clearRedactState();
}
