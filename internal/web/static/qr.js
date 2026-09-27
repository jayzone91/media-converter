let qrPreviewTimeout = null;
let qrPreviewController = null;

let qrLogoDataURL = "";
let lastGeneratedQRSVG = "";

export function setupQRUI() {
  setupQRTypes();
  setupQRStyles();
  setupQRColorMode();
  setupQRLivePreview();
  setupQRColorLabels();
  setupQRLogo();
  setupQRDownloads();
}

function setupQRTypes() {
  const buttons = document.querySelectorAll(".qr-type");

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      buttons.forEach((candidate) => {
        candidate.classList.toggle("active", candidate === button);
      });

      renderQRFields(button.dataset.qrType);

      scheduleQRPreview();
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

        scheduleQRPreview();
      });
    });
  });
}

function setupQRColorMode() {
  const colorModes = document.querySelectorAll('input[name="qr_color_mode"]');

  const gradientSettings = document.getElementById("qr-gradient-settings");

  if (!gradientSettings) {
    return;
  }

  colorModes.forEach((mode) => {
    mode.addEventListener("change", () => {
      const selected = document.querySelector(
        'input[name="qr_color_mode"]:checked',
      );

      gradientSettings.hidden = selected?.value !== "gradient";

      scheduleQRPreview();
    });
  });
}

function setupQRLivePreview() {
  const panel = document.querySelector('[data-panel="qr"]');

  if (!panel) {
    return;
  }

  panel.addEventListener("input", (event) => {
    if (event.target.matches("input, textarea, select")) {
      scheduleQRPreview();
    }
  });

  panel.addEventListener("change", (event) => {
    if (event.target.matches("input, textarea, select")) {
      scheduleQRPreview();
    }
  });

  const generateButton = document.getElementById("qr-generate");

  generateButton?.addEventListener("click", () => {
    generateQRPreview();
  });
}

function setupQRColorLabels() {
  const ids = [
    "qr-foreground",
    "qr-background",
    "qr-gradient-start",
    "qr-gradient-end",
  ];

  ids.forEach((id) => {
    const input = document.getElementById(id);

    if (!input) {
      return;
    }

    updateQRColorLabel(input);

    input.addEventListener("input", () => {
      updateQRColorLabel(input);
    });
  });
}

function updateQRColorLabel(input) {
  const code = input.parentElement?.querySelector("code");

  if (code) {
    code.textContent = input.value.toUpperCase();
  }
}

function setupQRLogo() {
  const input = document.getElementById("qr-logo");

  if (!input) {
    return;
  }

  /*
   * HTML stammt noch aus dem bestehenden Template.
   * Wir begrenzen die Dateitypen hier zusätzlich clientseitig.
   */
  input.accept = "image/png,image/jpeg,image/webp";

  const label = input.closest(".logo-upload");

  const title = label?.querySelector("strong");

  const subtitle = label?.querySelector("small");

  if (subtitle) {
    subtitle.textContent = "PNG, JPEG oder WEBP · max. 2 MiB";
  }

  input.addEventListener("change", async () => {
    const file = input.files?.[0];

    if (!file) {
      qrLogoDataURL = "";

      if (title) {
        title.textContent = "Logo auswählen";
      }

      scheduleQRPreview();
      return;
    }

    const allowedTypes = ["image/png", "image/jpeg", "image/webp"];

    if (!allowedTypes.includes(file.type)) {
      input.value = "";
      qrLogoDataURL = "";

      if (title) {
        title.textContent = "Ungültiges Dateiformat";
      }

      showQRStatus("Fehler");

      return;
    }

    if (file.size > 2 * 1024 * 1024) {
      input.value = "";
      qrLogoDataURL = "";

      if (title) {
        title.textContent = "Logo ist größer als 2 MiB";
      }

      showQRStatus("Fehler");

      return;
    }

    try {
      qrLogoDataURL = await readFileAsDataURL(file);

      if (title) {
        title.textContent = file.name;
      }

      scheduleQRPreview();
    } catch (error) {
      console.error("Logo konnte nicht gelesen werden:", error);

      input.value = "";
      qrLogoDataURL = "";

      if (title) {
        title.textContent = "Logo konnte nicht gelesen werden";
      }

      showQRStatus("Fehler");
    }
  });
}

function readFileAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();

    reader.addEventListener("load", () => {
      if (typeof reader.result === "string") {
        resolve(reader.result);

        return;
      }

      reject(new Error("Logo konnte nicht gelesen werden."));
    });

    reader.addEventListener("error", () => {
      reject(reader.error ?? new Error("Logo konnte nicht gelesen werden."));
    });

    reader.readAsDataURL(file);
  });
}

function setupQRDownloads() {
  const container = document.querySelector(".qr-download-actions");

  if (!container) {
    return;
  }

  const buttons = container.querySelectorAll("button");

  const pngButton = buttons[0];

  const svgButton = buttons[1];

  if (pngButton) {
    pngButton.textContent = "PNG herunterladen";

    pngButton.addEventListener("click", () => {
      downloadQRPNG();
    });
  }

  if (svgButton) {
    svgButton.textContent = "SVG herunterladen";

    svgButton.addEventListener("click", () => {
      downloadQRSVG();
    });
  }
}

function scheduleQRPreview() {
  clearTimeout(qrPreviewTimeout);

  qrPreviewTimeout = setTimeout(() => {
    generateQRPreview();
  }, 250);
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
  const colorMode =
    document.querySelector('input[name="qr_color_mode"]:checked')?.value ??
    "solid";

  return {
    type: getActiveQRType(),

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

      gradient_enabled: colorMode === "gradient",

      gradient_start:
        document.getElementById("qr-gradient-start")?.value ?? "#000000",

      gradient_end:
        document.getElementById("qr-gradient-end")?.value ?? "#3b82f6",

      module: getActiveQRStyle("dots"),

      corner_outer: getActiveQRStyle("corners-square"),

      corner_inner: getActiveQRStyle("corners-dot"),

      has_logo: qrLogoDataURL !== "",

      logo: qrLogoDataURL,
    },
  };
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

async function generateQRPreview() {
  const preview = document.getElementById("qr-preview");

  if (!preview) {
    return;
  }

  const errorLevel = document.getElementById("qr-error-level");

  const version = document.getElementById("qr-version");

  const request = buildQRRequest();

  if (!hasQRPayload(request)) {
    lastGeneratedQRSVG = "";

    showEmptyQRPreview(preview);

    setDownloadVisibility(false);

    if (errorLevel) {
      errorLevel.textContent = "L";
    }

    if (version) {
      version.textContent = "Auto";
    }

    showQRStatus("Bereit");

    return;
  }

  if (qrPreviewController) {
    qrPreviewController.abort();
  }

  const controller = new AbortController();

  qrPreviewController = controller;

  showQRStatus("Erzeuge …");

  try {
    const response = await fetch("/qr/generate", {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify(request),

      signal: controller.signal,
    });

    if (!response.ok) {
      const message = await response.text();

      throw new Error(message.trim() || "QR Code konnte nicht erzeugt werden.");
    }

    const result = await response.json();

    lastGeneratedQRSVG = result.svg;

    preview.innerHTML = result.svg;

    const svg = preview.querySelector("svg");

    if (svg) {
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

    setDownloadVisibility(true);

    showQRStatus("Aktuell");
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      return;
    }

    console.error("QR preview failed:", error);

    lastGeneratedQRSVG = "";

    setDownloadVisibility(false);

    showQRStatus("Fehler");

    showQRPreviewError(
      preview,
      error instanceof Error
        ? error.message
        : "QR Code konnte nicht erzeugt werden.",
    );
  } finally {
    if (qrPreviewController === controller) {
      qrPreviewController = null;
    }
  }
}

function showQRStatus(text) {
  const status = document.querySelector(".qr-status");

  if (status) {
    status.textContent = text;
  }
}

function setDownloadVisibility(visible) {
  const container = document.querySelector(".qr-download-actions");

  if (container) {
    container.hidden = !visible;
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

function showQRPreviewError(preview, message) {
  preview.innerHTML = `
    <div class="qr-preview-placeholder">
      <strong>
        Keine Vorschau
      </strong>

      <span>
        ${escapeHTML(message)}
      </span>
    </div>
  `;
}

function escapeHTML(value) {
  const element = document.createElement("div");

  element.textContent = value;

  return element.innerHTML;
}

function downloadQRSVG() {
  if (!lastGeneratedQRSVG) {
    return;
  }

  const blob = new Blob([lastGeneratedQRSVG], {
    type: "image/svg+xml;charset=utf-8",
  });

  downloadBlob(blob, "qr-code.svg");
}

async function downloadQRPNG() {
  if (!lastGeneratedQRSVG) {
    return;
  }

  try {
    const svgBlob = new Blob([lastGeneratedQRSVG], {
      type: "image/svg+xml;charset=utf-8",
    });

    const svgURL = URL.createObjectURL(svgBlob);

    try {
      const image = await loadImage(svgURL);

      const size = 1200;

      const canvas = document.createElement("canvas");

      canvas.width = size;

      canvas.height = size;

      const context = canvas.getContext("2d");

      if (!context) {
        throw new Error("Canvas konnte nicht erstellt werden.");
      }

      context.drawImage(image, 0, 0, size, size);

      const png = await canvasToBlob(canvas);

      downloadBlob(png, "qr-code.png");
    } finally {
      URL.revokeObjectURL(svgURL);
    }
  } catch (error) {
    console.error("PNG Export fehlgeschlagen:", error);

    showQRStatus("Exportfehler");
  }
}

function loadImage(url) {
  return new Promise((resolve, reject) => {
    const image = new Image();

    image.addEventListener("load", () => {
      resolve(image);
    });

    image.addEventListener("error", () => {
      reject(new Error("SVG konnte nicht für PNG gerendert werden."));
    });

    image.src = url;
  });
}

function canvasToBlob(canvas) {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error("PNG konnte nicht erzeugt werden."));

        return;
      }

      resolve(blob);
    }, "image/png");
  });
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

function renderQRFields(type) {
  const container = document.getElementById("qr-fields");

  if (!container) {
    return;
  }

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

          <select
            name="wifi_encryption"
          >
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

          <input
            name="event_title"
          />
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

          <input
            name="event_location"
          />
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
