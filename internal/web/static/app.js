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

function setupQRColorMode() {
  const colorModes = document.querySelectorAll('input[name="qr_color_mode"]');

  const gradientSettings = document.getElementById("qr-gradient-settings");

  colorModes.forEach((mode) => {
    mode.addEventListener("change", () => {
      gradientSettings.hidden = mode.value !== "gradient" || !mode.checked;

      scheduleQRPreview();
    });
  });
}

let qrPreviewTimeout = null;
let qrPreviewRequest = null;

function setupQRLivePreview() {
  const panel = document.querySelector('[data-panel="qr"]');

  if (!panel) {
    return;
  }

  panel.addEventListener("input", () => {
    scheduleQRPreview();
  });

  panel.addEventListener("change", () => {
    scheduleQRPreview();
  });

  document.querySelectorAll(".qr-type").forEach((button) => {
    button.addEventListener("click", () => {
      scheduleQRPreview();
    });
  });

  document.querySelectorAll(".style-option").forEach((button) => {
    button.addEventListener("click", () => {
      scheduleQRPreview();
    });
  });

  const generateButton = document.getElementById("qr-generate");

  generateButton?.addEventListener("click", () => {
    generateQRPreview();
  });
}

function scheduleQRPreview() {
  clearTimeout(qrPreviewTimeout);

  qrPreviewTimeout = setTimeout(() => {
    generateQRPreview();
  }, 250);
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

        <input
          name="vcard_firstname"
          autocomplete="given-name"
        />
      </label>

      <label class="field">
        <span>Nachname</span>

        <input
          name="vcard_lastname"
          autocomplete="family-name"
        />
      </label>
    </div>

    <div class="field-grid">
      <label class="field">
        <span>Firma</span>

        <input
          name="vcard_company"
          autocomplete="organization"
        />
      </label>

      <label class="field">
        <span>Position</span>

        <input
          name="vcard_position"
          autocomplete="organization-title"
        />
      </label>
    </div>

    <div class="field-grid">
      <label class="field">
        <span>Telefon (Arbeit)</span>

        <input
          type="tel"
          name="vcard_phone_work"
        />
      </label>

      <label class="field">
        <span>Telefon (Privat)</span>

        <input
          type="tel"
          name="vcard_phone_home"
        />
      </label>
    </div>

    <div class="field-grid">
      <label class="field">
        <span>Mobil (Arbeit)</span>

        <input
          type="tel"
          name="vcard_mobile_work"
        />
      </label>

      <label class="field">
        <span>Mobil (Privat)</span>

        <input
          type="tel"
          name="vcard_mobile_home"
        />
      </label>
    </div>

    <div class="field-grid">
      <label class="field">
        <span>Fax (Arbeit)</span>

        <input
          type="tel"
          name="vcard_fax_work"
        />
      </label>

      <label class="field">
        <span>E-Mail</span>

        <input
          type="email"
          name="vcard_email"
          autocomplete="email"
        />
      </label>
    </div>

    <label class="field">
      <span>Webseite</span>

      <input
        type="url"
        name="vcard_website"
        placeholder="https://example.com"
      />
    </label>

    <label class="field">
      <span>Straße</span>

      <input
        name="vcard_street"
        autocomplete="street-address"
      />
    </label>

    <div class="field-grid">
      <label class="field">
        <span>PLZ</span>

        <input
          name="vcard_postal_code"
          autocomplete="postal-code"
        />
      </label>

      <label class="field">
        <span>Stadt</span>

        <input
          name="vcard_city"
          autocomplete="address-level2"
        />
      </label>
    </div>

    <div class="field-grid">
      <label class="field">
        <span>Bundesland / Region</span>

        <input
          name="vcard_region"
          autocomplete="address-level1"
        />
      </label>

      <label class="field">
        <span>Land</span>

        <input
          name="vcard_country"
          autocomplete="country-name"
        />
      </label>
    </div>
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

function getActiveQRType() {
  return document.querySelector(".qr-type.active")?.dataset.qrType ?? "url";
}

function getActiveQRStyle(group) {
  return (
    document.querySelector(`[data-style-group="${group}"] .style-option.active`)
      ?.dataset.style ?? "square"
  );
}

function getQRField(name) {
  return document.querySelector(`[name="${name}"]`)?.value ?? "";
}

function getQRCheckbox(name) {
  return document.querySelector(`[name="${name}"]`)?.checked ?? false;
}

function buildQRRequest() {
  const type = getActiveQRType();

  const gradient =
    document.querySelector('input[name="qr_color_mode"]:checked')?.value ===
    "gradient";

  return {
    type,

    url: getQRField("qr_url"),
    text: getQRField("qr_text"),
    phone: getQRField("qr_phone"),

    wifi: {
      ssid: getQRField("wifi_ssid"),

      password: getQRField("wifi_password"),

      encryption: getQRField("wifi_encryption"),

      hidden: getQRCheckbox("wifi_hidden"),
    },

    vcard: {
      first_name: getQRField("vcard_firstname"),

      last_name: getQRField("vcard_lastname"),

      company: getQRField("vcard_company"),

      position: getQRField("vcard_position"),

      phone_work: getQRField("vcard_phone_work"),

      phone_home: getQRField("vcard_phone_home"),

      mobile_work: getQRField("vcard_mobile_work"),

      mobile_home: getQRField("vcard_mobile_home"),

      fax_work: getQRField("vcard_fax_work"),

      email: getQRField("vcard_email"),

      website: getQRField("vcard_website"),

      street: getQRField("vcard_street"),

      postal_code: getQRField("vcard_postal_code"),

      city: getQRField("vcard_city"),

      region: getQRField("vcard_region"),

      country: getQRField("vcard_country"),
    },

    event: {
      title: getQRField("event_title"),

      start: getQRField("event_start"),

      end: getQRField("event_end"),

      location: getQRField("event_location"),

      description: getQRField("event_description"),
    },

    style: {
      foreground: document.getElementById("qr-foreground")?.value ?? "#000000",

      background: document.getElementById("qr-background")?.value ?? "#ffffff",

      gradient_enabled: gradient,

      gradient_start:
        document.getElementById("qr-gradient-start")?.value ?? "#000000",

      gradient_end:
        document.getElementById("qr-gradient-end")?.value ?? "#3b82f6",

      module: getActiveQRStyle("dots"),

      corner_outer: getActiveQRStyle("corners-square"),

      corner_inner: getActiveQRStyle("corners-dot"),

      has_logo: false,
    },
  };
}

async function generateQRPreview() {
  const preview = document.getElementById("qr-preview");

  const errorLevel = document.getElementById("qr-error-level");

  const version = document.getElementById("qr-version");

  const status = document.querySelector(".qr-status");

  if (!preview) {
    return;
  }

  const request = buildQRRequest();

  if (!hasQRPayload(request)) {
    showEmptyQRPreview(preview);

    if (errorLevel) {
      errorLevel.textContent = "L";
    }

    if (version) {
      version.textContent = "Auto";
    }

    if (status) {
      status.textContent = "Bereit";
    }

    return;
  }

  if (qrPreviewRequest) {
    qrPreviewRequest.abort();
  }

  qrPreviewRequest = new AbortController();

  if (status) {
    status.textContent = "Aktualisiert …";
  }

  try {
    const response = await fetch("/qr/generate", {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify(request),

      signal: qrPreviewRequest.signal,
    });

    if (!response.ok) {
      const message = await response.text();

      throw new Error(message.trim() || "QR Code konnte nicht erzeugt werden.");
    }

    const result = await response.json();

    preview.innerHTML = result.svg;

    const svg = preview.querySelector("svg");

    if (svg) {
      svg.removeAttribute("width");

      svg.removeAttribute("height");

      svg.style.width = "100%";

      svg.style.height = "100%";

      svg.style.display = "block";
    }

    if (errorLevel) {
      errorLevel.textContent = result.error_correction;
    }

    if (version) {
      version.textContent = String(result.version);
    }

    if (status) {
      status.textContent = "Aktuell";
    }
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      return;
    }

    if (status) {
      status.textContent = "Ungültig";
    }
  } finally {
    qrPreviewRequest = null;
  }
}

function hasQRPayload(request) {
  switch (request.type) {
    case "url":
      return request.url.trim() !== "";

    case "text":
      return request.text.trim() !== "";

    case "phone":
      return request.phone.trim() !== "";

    case "wifi":
      return request.wifi.ssid.trim() !== "";

    case "vcard":
      return (
        request.vcard.first_name.trim() !== "" ||
        request.vcard.last_name.trim() !== ""
      );

    case "event":
      return (
        request.event.title.trim() !== "" &&
        request.event.start !== "" &&
        request.event.end !== ""
      );

    default:
      return false;
  }
}

function showEmptyQRPreview(preview) {
  preview.innerHTML = `
    <div class="qr-preview-placeholder">
      <div class="preview-icon">
        ▦
      </div>

      <strong>
        QR Code Vorschau
      </strong>

      <span>
        Gib links einen Inhalt ein.
      </span>
    </div>
  `;
}
