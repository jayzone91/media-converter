let mergeFiles = [];

export function renderPDFMerge(workspace, onBack) {
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

  setupBackButton(onBack);

  setupMergeWorkspace();
}

export function resetPDFMerge() {
  mergeFiles = [];
}

function setupBackButton(onBack) {
  const button = document.getElementById("pdf-tool-back");

  if (!button || typeof onBack !== "function") {
    return;
  }

  button.addEventListener("click", onBack);
}

function setupMergeWorkspace() {
  const input = document.getElementById("pdf-merge-files");

  const dropZone = document.getElementById("pdf-merge-drop-zone");

  const clearButton = document.getElementById("pdf-merge-clear");

  const submitButton = document.getElementById("pdf-merge-submit");

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
    mergePDFs();
  });
}

function setupMergeDropZone(dropZone) {
  ["dragenter", "dragover", "dragleave", "drop"].forEach((eventName) => {
    dropZone.addEventListener(eventName, (event) => {
      event.preventDefault();
      event.stopPropagation();
    });
  });

  ["dragenter", "dragover"].forEach((eventName) => {
    dropZone.addEventListener(eventName, () => {
      dropZone.classList.add("drag-over");
    });
  });

  ["dragleave", "drop"].forEach((eventName) => {
    dropZone.addEventListener(eventName, () => {
      dropZone.classList.remove("drag-over");
    });
  });

  dropZone.addEventListener("drop", (event) => {
    const files = event.dataTransfer?.files;

    if (!files?.length) {
      return;
    }

    addMergeFiles(Array.from(files));
  });
}

function addMergeFiles(files) {
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
    if (file.size > 512 * 1024 * 1024) {
      showMergeError(`${file.name} ist größer als 512 MiB.`);

      continue;
    }

    if (mergeFiles.length >= 50) {
      showMergeError(
        "Es können maximal 50 PDFs gleichzeitig zusammengefügt werden.",
      );

      break;
    }

    mergeFiles.push(createMergeFileEntry(file));
  }

  if (getMergeTotalSize() > 1024 * 1024 * 1024) {
    while (mergeFiles.length > 0 && getMergeTotalSize() > 1024 * 1024 * 1024) {
      mergeFiles.pop();
    }

    showMergeError("Die PDFs dürfen zusammen maximal 1 GiB groß sein.");
  }

  renderMergeFileList();
}

function createMergeFileEntry(file) {
  return {
    id: createID(),
    file,
  };
}

function createID() {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return [Date.now(), Math.random().toString(16).slice(2)].join("-");
}

function getMergeTotalSize() {
  return mergeFiles.reduce((total, entry) => total + entry.file.size, 0);
}

function renderMergeFileList() {
  const selection = document.getElementById("pdf-merge-selection");

  const list = document.getElementById("pdf-merge-list");

  const count = document.getElementById("pdf-merge-count");

  const submit = document.getElementById("pdf-merge-submit");

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

function setupMergeListActions() {
  document.querySelectorAll("[data-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      removeMergeFile(button.dataset.remove);
    });
  });

  document.querySelectorAll("[data-move-up]").forEach((button) => {
    button.addEventListener("click", () => {
      moveMergeFile(button.dataset.moveUp, -1);
    });
  });

  document.querySelectorAll("[data-move-down]").forEach((button) => {
    button.addEventListener("click", () => {
      moveMergeFile(button.dataset.moveDown, 1);
    });
  });
}

function removeMergeFile(id) {
  mergeFiles = mergeFiles.filter((entry) => entry.id !== id);

  renderMergeFileList();
}

function moveMergeFile(id, offset) {
  const index = mergeFiles.findIndex((entry) => entry.id === id);

  if (index < 0) {
    return;
  }

  const destination = index + offset;

  if (destination < 0 || destination >= mergeFiles.length) {
    return;
  }

  const [entry] = mergeFiles.splice(index, 1);

  mergeFiles.splice(destination, 0, entry);

  renderMergeFileList();
}

function setupMergeSorting() {
  const list = document.getElementById("pdf-merge-list");

  if (!list) {
    return;
  }

  let draggedID = null;

  list.querySelectorAll(".pdf-file-item").forEach((item) => {
    item.addEventListener("dragstart", () => {
      draggedID = item.dataset.fileId;

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

      if (!draggedID || draggedID === item.dataset.fileId) {
        return;
      }

      reorderMergeFiles(draggedID, item.dataset.fileId);
    });
  });
}

function clearDragTargets(list) {
  list.querySelectorAll(".pdf-file-item").forEach((item) => {
    item.classList.remove("drag-target");
  });
}

function reorderMergeFiles(sourceID, targetID) {
  const sourceIndex = mergeFiles.findIndex((entry) => entry.id === sourceID);

  const targetIndex = mergeFiles.findIndex((entry) => entry.id === targetID);

  if (sourceIndex < 0 || targetIndex < 0) {
    return;
  }

  const [source] = mergeFiles.splice(sourceIndex, 1);

  mergeFiles.splice(targetIndex, 0, source);

  renderMergeFileList();
}

async function mergePDFs() {
  if (mergeFiles.length < 2) {
    showMergeError("Bitte mindestens zwei PDFs auswählen.");

    return;
  }

  clearMergeError();

  const submit = document.getElementById("pdf-merge-submit");

  const progress = document.getElementById("pdf-merge-progress");

  if (submit) {
    submit.disabled = true;
  }

  if (progress) {
    progress.hidden = false;
  }

  const body = new FormData();

  mergeFiles.forEach((entry) => {
    body.append("files", entry.file, entry.file.name);
  });

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
      getDownloadFilename(response.headers.get("Content-Disposition")),
    );
  } catch (error) {
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

function showMergeError(message) {
  const error = document.getElementById("pdf-merge-error");

  if (!error) {
    return;
  }

  error.textContent = message;
  error.hidden = false;
}

function clearMergeError() {
  const error = document.getElementById("pdf-merge-error");

  if (!error) {
    return;
  }

  error.textContent = "";
  error.hidden = true;
}

function getDownloadFilename(contentDisposition) {
  if (!contentDisposition) {
    return "zusammengefuegt.pdf";
  }

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);

  if (utf8Match) {
    return decodeURIComponent(utf8Match[1]);
  }

  const filenameMatch = contentDisposition.match(/filename="([^"]+)"/i);

  if (filenameMatch) {
    return filenameMatch[1];
  }

  return "zusammengefuegt.pdf";
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);

  const link = document.createElement("a");

  link.href = url;
  link.download = filename;

  document.body.appendChild(link);

  link.click();
  link.remove();

  setTimeout(() => {
    URL.revokeObjectURL(url);
  }, 1000);
}

function formatBytes(bytes) {
  if (bytes === 0) {
    return "0 B";
  }

  const units = ["B", "KiB", "MiB", "GiB"];

  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );

  const value = bytes / 1024 ** index;

  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function escapeHTML(value) {
  const element = document.createElement("div");

  element.textContent = value;

  return element.innerHTML;
}
