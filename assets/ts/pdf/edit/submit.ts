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

interface PDFEditImageRequest {
  page: number;

  x: number;
  y: number;

  width: number;
}

interface PDFEditMetadata {
  upload_id: string;

  texts: PDFEditTextRequest[];

  images: PDFEditImageRequest[];
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

  button.disabled = !editorState.activeUpload || !hasEdits();
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

  const formData = buildEditFormData(upload.id);

  if (!formData) {
    return;
  }

  button.disabled = true;

  const originalText = button.textContent;

  button.textContent = "PDF wird erstellt …";

  try {
    const response = await fetch("/pdf/edit", {
      method: "POST",

      body: formData,
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

function buildEditFormData(uploadID: string): FormData | null {
  const texts = buildTextRequest();

  const images = buildImageRequest();

  if (texts.length === 0 && images.metadata.length === 0) {
    return null;
  }

  const metadata: PDFEditMetadata = {
    upload_id: uploadID,

    texts,

    images: images.metadata,
  };

  const body = new FormData();

  body.append("metadata", JSON.stringify(metadata));

  for (const image of images.files) {
    body.append("image", image.file, image.name);
  }

  return body;
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

function buildImageRequest(): {
  metadata: PDFEditImageRequest[];
  files: Array<{
    file: File;
    name: string;
  }>;
} {
  const metadata: PDFEditImageRequest[] = [];

  const files: Array<{
    file: File;
    name: string;
  }> = [];

  for (const [page, objects] of editorState.pageImageObjects) {
    for (const object of objects) {
      metadata.push({
        page: page + 1,

        x: object.x,
        y: object.y,

        width: object.width,
      });

      files.push({
        file: object.file,
        name: object.name,
      });
    }
  }

  return {
    metadata,
    files,
  };
}

function hasEdits(): boolean {
  for (const objects of editorState.pageTextObjects.values()) {
    if (objects.length > 0) {
      return true;
    }
  }

  for (const objects of editorState.pageImageObjects.values()) {
    if (objects.length > 0) {
      return true;
    }
  }

  return false;
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
