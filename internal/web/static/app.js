document.addEventListener("DOMContentLoaded", async () => {
  setupTabs();
  setupToolSelections();
  setupConverter();

  try {
    const qr = await import("/static/qr.js?v=1");
    qr.setupQRUI();
  } catch (error) {
    console.error("QR-Modul konnte nicht geladen werden:", error);
  }
});

function setupConverter() {
  const form = document.getElementById("conversion-form");
  const fileInput = document.getElementById("file");
  const dropZone = document.querySelector(".drop-zone");
  const options = document.getElementById("conversion-options");
  const overlay = document.getElementById("conversion-overlay");
  const errorBox = document.getElementById("conversion-error");

  if (!form || !fileInput || !dropZone || !options || !overlay || !errorBox) {
    return;
  }

  setupDragAndDrop(dropZone, fileInput, errorBox);

  form.addEventListener("submit", async (event) => {
    event.preventDefault();

    if (!form.reportValidity()) {
      return;
    }

    const uploadID = form.querySelector('input[name="upload_id"]');

    const target = form.querySelector('select[name="target"]');

    if (!uploadID || !target) {
      return;
    }

    errorBox.hidden = true;
    errorBox.textContent = "";

    overlay.hidden = false;

    const submitButton = form.querySelector('button[type="submit"]');

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
        const message = await response.text();

        throw new Error(message.trim() || "Konvertierung fehlgeschlagen.");
      }

      const blob = await response.blob();

      const filename = getDownloadFilename(
        response.headers.get("Content-Disposition"),
      );

      downloadBlob(blob, filename);

      resetForm(form, fileInput, options);
    } catch (error) {
      errorBox.textContent =
        error instanceof Error
          ? error.message
          : "Konvertierung fehlgeschlagen.";

      errorBox.hidden = false;

      resetForm(form, fileInput, options);
    } finally {
      overlay.hidden = true;

      if (submitButton) {
        submitButton.disabled = false;
      }
    }
  });
}

function setupDragAndDrop(dropZone, fileInput, errorBox) {
  const preventDefaults = (event) => {
    event.preventDefault();
    event.stopPropagation();
  };

  ["dragenter", "dragover", "dragleave", "drop"].forEach((eventName) => {
    dropZone.addEventListener(eventName, preventDefaults);
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

    if (!files || files.length === 0) {
      return;
    }

    if (files.length > 1) {
      errorBox.textContent = "Bitte nur eine Datei gleichzeitig auswählen.";

      errorBox.hidden = false;

      return;
    }

    errorBox.hidden = true;
    errorBox.textContent = "";

    const transfer = new DataTransfer();

    transfer.items.add(files[0]);

    fileInput.files = transfer.files;

    fileInput.dispatchEvent(
      new Event("change", {
        bubbles: true,
      }),
    );
  });

  document.addEventListener("dragover", (event) => {
    event.preventDefault();
  });

  document.addEventListener("drop", (event) => {
    if (!dropZone.contains(event.target)) {
      event.preventDefault();
    }
  });
}

function getDownloadFilename(contentDisposition) {
  if (!contentDisposition) {
    return "converted-file";
  }

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);

  if (utf8Match) {
    return decodeURIComponent(utf8Match[1]);
  }

  const filenameMatch = contentDisposition.match(/filename="([^"]+)"/i);

  if (filenameMatch) {
    return filenameMatch[1];
  }

  return "converted-file";
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

function resetForm(form, fileInput, options) {
  form.reset();

  fileInput.value = "";

  options.innerHTML = `
    <div class="empty-state">
      Wähle eine Datei aus, um die verfügbaren
      Zielformate anzuzeigen.
    </div>
  `;
}

function setupTabs() {
  const buttons = document.querySelectorAll(".tab-button");

  const panels = document.querySelectorAll(".tab-panel");

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      const target = button.dataset.tab;

      buttons.forEach((candidate) => {
        candidate.classList.toggle("active", candidate === button);
      });

      panels.forEach((panel) => {
        const active = panel.dataset.panel === target;

        panel.classList.toggle("active", active);

        panel.hidden = !active;
      });
    });
  });
}

function setupToolSelections() {
  document.querySelectorAll("[data-pdf-tool]").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelectorAll("[data-pdf-tool]").forEach((candidate) => {
        candidate.classList.remove("selected");
      });

      button.classList.add("selected");
    });
  });
}
