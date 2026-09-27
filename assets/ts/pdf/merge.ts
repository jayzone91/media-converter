import { downloadBlob, getDownloadFilename } from "../shared/download.ts";

interface MergeFile {
  id: string;
  file: File;
}

const MAX_FILES = 50;
const MAX_FILE_SIZE = 512 * 1024 * 1024;
const MAX_TOTAL_SIZE = 1024 * 1024 * 1024;

let mergeFiles: MergeFile[] = [];

export function renderPDFMerge(
  workspace: HTMLElement,
  onBack: () => void,
): void {
  mergeFiles = [];

  workspace.innerHTML = `
    <div class="pdf-workspace-navigation">
      <button
        id="pdf-tool-back"
        class="pdf-back-button"
        type="button"
      >
        <span aria-hidden="true">
          ←
        </span>

        <span>
          Zurück zu PDF-Werkzeugen
        </span>
      </button>
    </div>

    <div class="pdf-workspace-header">
      <div>
        <span class="eyebrow">
          PDF
        </span>

        <h3>
          PDFs zusammenfügen
        </h3>

        <p>
          Wähle mindestens zwei PDF-Dateien.
          Die Reihenfolge kann anschließend
          per Drag & Drop oder mit den
          Pfeiltasten geändert werden.
        </p>
      </div>
    </div>

    <label
      class="pdf-drop-zone"
      id="pdf-merge-drop-zone"
      for="pdf-merge-files"
    >
      <span class="pdf-drop-icon">
        ↑
      </span>

      <strong>
        PDFs auswählen
      </strong>

      <span>
        oder mehrere Dateien hier ablegen
      </span>

      <small>
        Max. 50 Dateien · 512 MiB pro Datei · 1 GiB gesamt
      </small>

      <input
        id="pdf-merge-files"
        type="file"
        accept="application/pdf,.pdf"
        multiple
        hidden
      />
    </label>

    <div
      id="pdf-merge-error"
      class="pdf-message pdf-message-error"
      hidden
    ></div>

    <section
      id="pdf-merge-selection"
      class="pdf-selection"
      hidden
    >
      <header class="pdf-selection-header">
        <div>
          <strong>
            Reihenfolge
          </strong>

          <span id="pdf-merge-count">
            0 Dateien
          </span>
        </div>

        <button
          id="pdf-merge-clear"
          class="pdf-button-secondary"
          type="button"
        >
          Alle entfernen
        </button>
      </header>

      <div
        id="pdf-merge-list"
        class="pdf-file-list"
      ></div>

      <div class="pdf-action-row">
        <div>
          <strong>
            Ausgabe
          </strong>

          <span>
            zusammengefuegt.pdf
          </span>
        </div>

        <button
          id="pdf-merge-submit"
          class="pdf-button-primary"
          type="button"
          disabled
        >
          PDFs zusammenfügen
        </button>
      </div>
    </section>

    <div
      id="pdf-merge-progress"
      class="pdf-progress"
      hidden
    >
      <span class="small-spinner"></span>

      <div>
        <strong>
          PDFs werden zusammengefügt …
        </strong>

        <span>
          Bitte diese Seite geöffnet lassen.
        </span>
      </div>
    </div>
  `;

  const backButton =
    document.querySelector<HTMLButtonElement>("#pdf-tool-back");

  backButton?.addEventListener("click", onBack);

  setupMergeWorkspace();
}

export function resetPDFMerge(): void {
  mergeFiles = [];
}

function setupMergeWorkspace(): void {
  const input = document.querySelector<HTMLInputElement>("#pdf-merge-files");

  const dropZone = document.querySelector<HTMLElement>("#pdf-merge-drop-zone");

  const clearButton =
    document.querySelector<HTMLButtonElement>("#pdf-merge-clear");

  const submitButton =
    document.querySelector<HTMLButtonElement>("#pdf-merge-submit");

  if (!input || !dropZone || !clearButton || !submitButton) {
    return;
  }

  input.addEventListener("change", () => {
    if (input.files?.length) {
      addMergeFiles(Array.from(input.files));
    }

    input.value = "";
  });

  setupMergeDropZone(dropZone);

  clearButton.addEventListener("click", () => {
    mergeFiles = [];

    renderMergeFileList();
  });

  submitButton.addEventListener("click", () => {
    void mergePDFs();
  });
}

function setupMergeDropZone(dropZone: HTMLElement): void {
  const dragEvents = ["dragenter", "dragover", "dragleave", "drop"] as const;

  for (const eventName of dragEvents) {
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

    addMergeFiles(Array.from(files));
  });
}

function addMergeFiles(files: File[]): void {
  clearMergeError();

  const PDFs = files.filter(
    (file) =>
      file.type === "application/pdf" ||
      file.name.toLowerCase().endsWith(".pdf"),
  );

  if (PDFs.length !== files.length) {
    showMergeError("Es können nur PDF-Dateien hinzugefügt werden.");
  }

  for (const file of PDFs) {
    if (file.size > MAX_FILE_SIZE) {
      showMergeError(`${file.name} ist größer als 512 MiB.`);

      continue;
    }

    if (mergeFiles.length >= MAX_FILES) {
      showMergeError(
        "Es können maximal 50 PDFs gleichzeitig zusammengefügt werden.",
      );

      break;
    }

    mergeFiles.push({
      id: createID(),
      file,
    });
  }

  if (getMergeTotalSize() > MAX_TOTAL_SIZE) {
    while (mergeFiles.length > 0 && getMergeTotalSize() > MAX_TOTAL_SIZE) {
      mergeFiles.pop();
    }

    showMergeError("Die PDFs dürfen zusammen maximal 1 GiB groß sein.");
  }

  renderMergeFileList();
}

function createID(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return [Date.now(), Math.random().toString(16).slice(2)].join("-");
}

function getMergeTotalSize(): number {
  return mergeFiles.reduce((total, entry) => total + entry.file.size, 0);
}

function renderMergeFileList(): void {
  const selection = document.querySelector<HTMLElement>("#pdf-merge-selection");

  const list = document.querySelector<HTMLElement>("#pdf-merge-list");

  const count = document.querySelector<HTMLElement>("#pdf-merge-count");

  const submit = document.querySelector<HTMLButtonElement>("#pdf-merge-submit");

  if (!selection || !list || !count || !submit) {
    return;
  }

  selection.hidden = mergeFiles.length === 0;

  count.textContent =
    mergeFiles.length === 1 ? "1 Datei" : `${mergeFiles.length} Dateien`;

  submit.disabled = mergeFiles.length < 2;

  list.innerHTML = mergeFiles
    .map(
      (entry, index) => `
          <article
            class="pdf-file-item"
            draggable="true"
            data-file-id="${entry.id}"
          >
            <span
              class="pdf-file-drag"
              title="Verschieben"
            >
              ⠿
            </span>

            <span class="pdf-file-index">
              ${index + 1}
            </span>

            <div class="pdf-file-info">
              <strong
                title="${escapeHTML(entry.file.name)}"
              >
                ${escapeHTML(entry.file.name)}
              </strong>

              <span>
                ${formatBytes(entry.file.size)}
              </span>
            </div>

            <div class="pdf-file-actions">
              <button
                type="button"
                data-move-up="${entry.id}"
                title="Nach oben"
                ${index === 0 ? "disabled" : ""}
              >
                ↑
              </button>

              <button
                type="button"
                data-move-down="${entry.id}"
                title="Nach unten"
                ${index === mergeFiles.length - 1 ? "disabled" : ""}
              >
                ↓
              </button>

              <button
                type="button"
                data-remove="${entry.id}"
                title="Entfernen"
              >
                ×
              </button>
            </div>
          </article>
        `,
    )
    .join("");

  setupMergeListActions();
  setupMergeSorting();
}

function setupMergeListActions(): void {
  document
    .querySelectorAll<HTMLButtonElement>("[data-remove]")
    .forEach((button) => {
      button.addEventListener("click", () => {
        const id = button.dataset.remove;

        if (id) {
          removeMergeFile(id);
        }
      });
    });

  document
    .querySelectorAll<HTMLButtonElement>("[data-move-up]")
    .forEach((button) => {
      button.addEventListener("click", () => {
        const id = button.dataset.moveUp;

        if (id) {
          moveMergeFile(id, -1);
        }
      });
    });

  document
    .querySelectorAll<HTMLButtonElement>("[data-move-down]")
    .forEach((button) => {
      button.addEventListener("click", () => {
        const id = button.dataset.moveDown;

        if (id) {
          moveMergeFile(id, 1);
        }
      });
    });
}

function removeMergeFile(id: string): void {
  mergeFiles = mergeFiles.filter((entry) => entry.id !== id);

  renderMergeFileList();
}

function moveMergeFile(id: string, offset: number): void {
  const index = mergeFiles.findIndex((entry) => entry.id === id);

  if (index < 0) {
    return;
  }

  const destination = index + offset;

  if (destination < 0 || destination >= mergeFiles.length) {
    return;
  }

  const entry = mergeFiles[index];

  if (!entry) {
    return;
  }

  mergeFiles.splice(index, 1);

  mergeFiles.splice(destination, 0, entry);

  renderMergeFileList();
}

function setupMergeSorting(): void {
  const list = document.querySelector<HTMLElement>("#pdf-merge-list");

  if (!list) {
    return;
  }

  let draggedID: string | null = null;

  list.querySelectorAll<HTMLElement>(".pdf-file-item").forEach((item) => {
    item.addEventListener("dragstart", () => {
      draggedID = item.dataset.fileId ?? null;

      item.classList.add("dragging");
    });

    item.addEventListener("dragend", () => {
      draggedID = null;

      item.classList.remove("dragging");

      clearDragTargets(list);
    });

    item.addEventListener("dragover", (event) => {
      event.preventDefault();

      if (!draggedID || draggedID === item.dataset.fileId) {
        return;
      }

      clearDragTargets(list);

      item.classList.add("drag-target");
    });

    item.addEventListener("dragleave", () => {
      item.classList.remove("drag-target");
    });

    item.addEventListener("drop", (event) => {
      event.preventDefault();

      item.classList.remove("drag-target");

      const targetID = item.dataset.fileId;

      if (!draggedID || !targetID || draggedID === targetID) {
        return;
      }

      reorderMergeFiles(draggedID, targetID);
    });
  });
}

function clearDragTargets(list: HTMLElement): void {
  list.querySelectorAll<HTMLElement>(".pdf-file-item").forEach((item) => {
    item.classList.remove("drag-target");
  });
}

function reorderMergeFiles(sourceID: string, targetID: string): void {
  const sourceIndex = mergeFiles.findIndex((entry) => entry.id === sourceID);

  const targetIndex = mergeFiles.findIndex((entry) => entry.id === targetID);

  if (sourceIndex < 0 || targetIndex < 0) {
    return;
  }

  const source = mergeFiles[sourceIndex];

  if (!source) {
    return;
  }

  mergeFiles.splice(sourceIndex, 1);

  mergeFiles.splice(targetIndex, 0, source);

  renderMergeFileList();
}

async function mergePDFs(): Promise<void> {
  if (mergeFiles.length < 2) {
    showMergeError("Bitte mindestens zwei PDFs auswählen.");

    return;
  }

  clearMergeError();

  const submit = document.querySelector<HTMLButtonElement>("#pdf-merge-submit");

  const progress = document.querySelector<HTMLElement>("#pdf-merge-progress");

  if (submit) {
    submit.disabled = true;
  }

  if (progress) {
    progress.hidden = false;
  }

  const body = new FormData();

  for (const entry of mergeFiles) {
    body.append("files", entry.file, entry.file.name);
  }

  try {
    const response = await fetch("/pdf/merge", {
      method: "POST",
      body,
    });

    if (!response.ok) {
      const message = await response.text();

      throw new Error(
        message.trim() || "PDFs konnten nicht zusammengefügt werden.",
      );
    }

    const blob = await response.blob();

    downloadBlob(
      blob,
      getDownloadFilename(
        response.headers.get("Content-Disposition"),
        "zusammengefuegt.pdf",
      ),
    );
  } catch (error: unknown) {
    showMergeError(
      error instanceof Error
        ? error.message
        : "PDFs konnten nicht zusammengefügt werden.",
    );
  } finally {
    if (progress) {
      progress.hidden = true;
    }

    if (submit) {
      submit.disabled = mergeFiles.length < 2;
    }
  }
}

function showMergeError(message: string): void {
  const error = document.querySelector<HTMLElement>("#pdf-merge-error");

  if (!error) {
    return;
  }

  error.textContent = message;
  error.hidden = false;
}

function clearMergeError(): void {
  const error = document.querySelector<HTMLElement>("#pdf-merge-error");

  if (!error) {
    return;
  }

  error.textContent = "";
  error.hidden = true;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) {
    return "0 B";
  }

  const units = ["B", "KiB", "MiB", "GiB"] as const;

  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );

  const unit = units[index] ?? "B";

  const value = bytes / 1024 ** index;

  return `${value.toFixed(index === 0 ? 0 : 1)} ${unit}`;
}

function escapeHTML(value: string): string {
  const element = document.createElement("div");

  element.textContent = value;

  return element.innerHTML;
}
