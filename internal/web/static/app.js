document.addEventListener("DOMContentLoaded", () => {
  const form = document.getElementById("conversion-form");
  const fileInput = document.getElementById("file");
  const dropZone = document.querySelector(".drop-zone");
  const options = document.getElementById("conversion-options");
  const overlay = document.getElementById("conversion-overlay");
  const errorBox = document.getElementById("conversion-error");
  setupTabs();
  setupToolSelections();
  setupQRUI();

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
});

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

    const file = files[0];

    errorBox.hidden = true;
    errorBox.textContent = "";

    const transfer = new DataTransfer();

    transfer.items.add(file);

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

  URL.revokeObjectURL(url);
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

function setupQRUI() {
  setupQRTypes();
  setupQRStyles();

  const gradientToggle = document.getElementById("qr-gradient-enabled");

  const gradientSettings = document.getElementById("qr-gradient-settings");

  gradientToggle?.addEventListener("change", () => {
    gradientSettings.hidden = !gradientToggle.checked;
  });
}

function setupQRTypes() {
  const buttons = document.querySelectorAll(".qr-type");

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      buttons.forEach((candidate) => {
        candidate.classList.toggle("active", candidate === button);
      });

      renderQRFields(button.dataset.qrType);
    });
  });
}

function setupQRStyles() {
  document.querySelectorAll("[data-style-group]").forEach((group) => {
    const options = group.querySelectorAll(".style-option");

    options.forEach((option) => {
      option.addEventListener("click", () => {
        options.forEach((candidate) => {
          candidate.classList.toggle("active", candidate === option);
        });
      });
    });
  });
}

function renderQRFields(type) {
  const container = document.getElementById("qr-fields");

  switch (type) {
    case "url":
      container.innerHTML = `
        <label class="field">
          <span>URL</span>

          <input
            type="url"
            name="qr_url"
            placeholder="https://example.com"
          />
        </label>
      `;
      break;

    case "text":
      container.innerHTML = `
        <label class="field">
          <span>Text</span>

          <textarea
            name="qr_text"
            rows="5"
            placeholder="Text eingeben"
          ></textarea>
        </label>
      `;
      break;

    case "phone":
      container.innerHTML = `
        <label class="field">
          <span>Telefonnummer</span>

          <input
            type="tel"
            name="qr_phone"
            placeholder="+49 ..."
          />
        </label>
      `;
      break;

    case "wifi":
      container.innerHTML = `
        <label class="field">
          <span>SSID</span>

          <input
            type="text"
            name="wifi_ssid"
          />
        </label>

        <label class="field">
          <span>Passwort</span>

          <input
            type="password"
            name="wifi_password"
          />
        </label>

        <label class="field">
          <span>Verschlüsselung</span>

          <select name="wifi_encryption">
            <option value="WPA">
              WPA / WPA2 / WPA3
            </option>

            <option value="WEP">
              WEP
            </option>

            <option value="nopass">
              Offen
            </option>
          </select>
        </label>

        <label class="toggle-row">
          <input
            type="checkbox"
            name="wifi_hidden"
          />

          <span>
            Verstecktes WLAN
          </span>
        </label>
      `;
      break;

    case "vcard":
      container.innerHTML = `
        <div class="field-grid">
          <label class="field">
            <span>Vorname</span>
            <input name="vcard_firstname" />
          </label>

          <label class="field">
            <span>Nachname</span>
            <input name="vcard_lastname" />
          </label>
        </div>

        <label class="field">
          <span>Telefonnummer</span>
          <input
            type="tel"
            name="vcard_phone"
          />
        </label>

        <label class="field">
          <span>E-Mail</span>
          <input
            type="email"
            name="vcard_email"
          />
        </label>

        <label class="field">
          <span>Firma</span>
          <input name="vcard_company" />
        </label>
      `;
      break;

    case "event":
      container.innerHTML = `
        <label class="field">
          <span>Titel</span>
          <input name="event_title" />
        </label>

        <div class="field-grid">
          <label class="field">
            <span>Beginn</span>
            <input
              type="datetime-local"
              name="event_start"
            />
          </label>

          <label class="field">
            <span>Ende</span>
            <input
              type="datetime-local"
              name="event_end"
            />
          </label>
        </div>

        <label class="field">
          <span>Ort</span>
          <input name="event_location" />
        </label>

        <label class="field">
          <span>Beschreibung</span>

          <textarea
            name="event_description"
            rows="4"
          ></textarea>
        </label>
      `;
      break;
  }
}
