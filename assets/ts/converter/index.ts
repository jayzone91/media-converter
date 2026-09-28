import { readAPIError } from "../shared/api-error.ts";
import { downloadURL } from "../shared/download.ts";

interface ConversionResponse {
  download_url: string;
  filename: string;
}

export function setupConverter(): void {
  const form = document.querySelector<HTMLFormElement>("#conversion-form");

  const fileInput = document.querySelector<HTMLInputElement>("#file");

  const dropZone = document.querySelector<HTMLElement>(".drop-zone");

  const options = document.querySelector<HTMLElement>("#conversion-options");

  const overlay = document.querySelector<HTMLElement>("#conversion-overlay");

  const errorBox = document.querySelector<HTMLElement>("#conversion-error");

  if (!form || !fileInput || !dropZone || !options || !overlay || !errorBox) {
    return;
  }

  setupDragAndDrop(dropZone, fileInput, errorBox);

  form.addEventListener("submit", (event) => {
    void handleConversionSubmit(
      event,
      form,
      fileInput,
      options,
      overlay,
      errorBox,
    );
  });
}

async function handleConversionSubmit(
  event: SubmitEvent,
  form: HTMLFormElement,
  fileInput: HTMLInputElement,
  options: HTMLElement,
  overlay: HTMLElement,
  errorBox: HTMLElement,
): Promise<void> {
  event.preventDefault();

  if (!form.reportValidity()) {
    return;
  }

  const uploadID = form.querySelector<HTMLInputElement>(
    'input[name="upload_id"]',
  );

  const target = form.querySelector<HTMLSelectElement>('select[name="target"]');

  if (!uploadID || !target) {
    return;
  }

  clearError(errorBox);

  overlay.hidden = false;

  const submitButton = form.querySelector<HTMLButtonElement>(
    'button[type="submit"]',
  );

  if (submitButton) {
    submitButton.disabled = true;
  }

  const body = new URLSearchParams();

  body.set("upload_id", uploadID.value);

  body.set("target", target.value);

  try {
    const response = await fetch(form.action, {
      method: "POST",

      headers: {
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
      },

      body,
    });

    if (!response.ok) {
      const error = await readAPIError(
        response,
        "Konvertierung fehlgeschlagen.",
      );

      throw new Error(error.message);
    }

    const result = (await response.json()) as ConversionResponse;

    if (!result.download_url) {
      throw new Error("Der Download konnte nicht vorbereitet werden.");
    }

    downloadURL(result.download_url);

    resetForm(form, fileInput, options);
  } catch (error: unknown) {
    errorBox.textContent = getErrorMessage(
      error,
      "Konvertierung fehlgeschlagen.",
    );

    errorBox.hidden = false;

    resetForm(form, fileInput, options);
  } finally {
    overlay.hidden = true;

    if (submitButton) {
      submitButton.disabled = false;
    }
  }
}

function setupDragAndDrop(
  dropZone: HTMLElement,
  fileInput: HTMLInputElement,
  errorBox: HTMLElement,
): void {
  const preventDefaults = (event: Event): void => {
    event.preventDefault();
    event.stopPropagation();
  };

  const dragEvents = ["dragenter", "dragover", "dragleave", "drop"] as const;

  for (const eventName of dragEvents) {
    dropZone.addEventListener(eventName, preventDefaults);
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

    if (!files || files.length === 0) {
      return;
    }

    if (files.length > 20) {
      errorBox.textContent = "Maximal 20 Dateien gleichzeitig.";

      errorBox.hidden = false;

      return;
    }

    clearError(errorBox);

    const transfer = new DataTransfer();

    for (const file of files) {
      transfer.items.add(file);
    }

    fileInput.files = transfer.files;

    fileInput.dispatchEvent(
      new Event("change", {
        bubbles: true,
      }),
    );
  });

  document.addEventListener("dragover", (event: DragEvent) => {
    event.preventDefault();
  });

  document.addEventListener("drop", (event: DragEvent) => {
    const target = event.target;

    if (!(target instanceof Node) || !dropZone.contains(target)) {
      event.preventDefault();
    }
  });
}

function clearError(errorBox: HTMLElement): void {
  errorBox.hidden = true;
  errorBox.textContent = "";
}

function resetForm(
  form: HTMLFormElement,
  fileInput: HTMLInputElement,
  options: HTMLElement,
): void {
  form.reset();

  fileInput.value = "";

  options.innerHTML = `
    <div class="empty-state">
      Wähle eine oder mehrere Dateien gleichen Typs aus,
      um die verfügbaren Zielformate anzuzeigen.
    </div>
  `;
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) {
    return error.message;
  }

  return fallback;
}
