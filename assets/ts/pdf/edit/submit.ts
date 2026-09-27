import { editorState } from "./state.ts";

interface PDFEditDownloadResponse {
  download_url: string;
  filename: string;
}

interface PDFEditTextRequest {
  page: number;
  text: string;
  x: number;
  y: number;
  size: number;
  color: string;
}

export function setupPDFEditSubmit(): boolean {
  const root = editorState.root;

  if (!root) {
    return false;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-edit-submit");

  if (!button) {
    return false;
  }

  button.addEventListener("click", () => {
    void submitPDFEdit();
  });

  updatePDFEditSubmitState();

  return true;
}

export function updatePDFEditSubmitState(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-edit-submit");

  if (!button) {
    return;
  }

  button.disabled = !editorState.activeUpload || countTextObjects() === 0;
}

async function submitPDFEdit(): Promise<void> {
  const root = editorState.root;

  const upload = editorState.activeUpload;

  if (!root || !upload) {
    return;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-edit-submit");

  const error = root.querySelector<HTMLElement>("#pdf-edit-error");

  if (!button) {
    return;
  }

  if (error) {
    error.hidden = true;
    error.textContent = "";
  }

  const texts = buildTextRequest();

  if (texts.length === 0) {
    return;
  }

  button.disabled = true;

  const originalText = button.textContent;

  button.textContent = "PDF wird erstellt …";

  try {
    const response = await fetch("/pdf/edit", {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        upload_id: upload.id,
        texts,
      }),
    });

    if (!response.ok) {
      const message = await response.text();

      throw new Error(message.trim() || "PDF konnte nicht erstellt werden.");
    }

    const data = (await response.json()) as unknown;

    if (!isDownloadResponse(data)) {
      throw new Error("Der Server hat eine ungültige Antwort geliefert.");
    }

    editorState.activeUpload = null;

    window.location.assign(data.download_url);
  } catch (caught: unknown) {
    if (error) {
      error.textContent = errorMessage(caught);

      error.hidden = false;
    }
  } finally {
    button.textContent = originalText;

    updatePDFEditSubmitState();
  }
}

function buildTextRequest(): PDFEditTextRequest[] {
  const texts: PDFEditTextRequest[] = [];

  for (const [page, objects] of editorState.pageTextObjects) {
    for (const object of objects) {
      if (object.text.trim() === "") {
        continue;
      }

      texts.push({
        page: page + 1,

        text: object.text,

        x: object.x,
        y: object.y,

        size: object.size,

        color: object.color,
      });
    }
  }

  return texts;
}

function countTextObjects(): number {
  let count = 0;

  for (const objects of editorState.pageTextObjects.values()) {
    count += objects.length;
  }

  return count;
}

function isDownloadResponse(value: unknown): value is PDFEditDownloadResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    typeof candidate.download_url === "string" &&
    typeof candidate.filename === "string"
  );
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return "PDF konnte nicht erstellt werden.";
}
