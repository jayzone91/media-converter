let activeTool = null;
let mergeModule = null;

export function setupPDFUI() {
  ensurePDFStyles();

  const panel = document.querySelector('[data-panel="pdf"]');

  if (!panel) {
    return;
  }

  const buttons = panel.querySelectorAll("[data-pdf-tool]");

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      openPDFTool(button.dataset.pdfTool);
    });
  });
}

function ensurePDFStyles() {
  if (document.querySelector("link[data-pdf-styles]")) {
    return;
  }

  const link = document.createElement("link");

  link.rel = "stylesheet";
  link.href = "/static/pdf.css?v=2";

  link.dataset.pdfStyles = "true";

  document.head.appendChild(link);
}

async function openPDFTool(tool) {
  const panel = document.querySelector('[data-panel="pdf"]');

  const grid = panel?.querySelector(".tool-grid");

  const workspace = document.getElementById("pdf-tool-workspace");

  if (!panel || !grid || !workspace) {
    return;
  }

  activeTool = tool;

  panel.querySelectorAll("[data-pdf-tool]").forEach((button) => {
    button.classList.toggle("selected", button.dataset.pdfTool === tool);
  });

  grid.hidden = true;

  workspace.hidden = false;

  renderWorkspaceLoading(workspace);

  switch (tool) {
    case "merge":
      await openMergeTool(workspace);

      break;

    default:
      renderComingSoon(workspace, tool);
  }
}

async function openMergeTool(workspace) {
  try {
    if (!mergeModule) {
      mergeModule = await import("/static/pdf-merge.js?v=1");
    }

    mergeModule.renderPDFMerge(workspace, closePDFTool);
  } catch (error) {
    console.error("PDF-Merge-Modul konnte nicht geladen werden:", error);

    renderWorkspaceError(
      workspace,
      "PDF zusammenfügen konnte nicht geladen werden.",
    );
  }
}

function closePDFTool() {
  const panel = document.querySelector('[data-panel="pdf"]');

  const grid = panel?.querySelector(".tool-grid");

  const workspace = document.getElementById("pdf-tool-workspace");

  if (!panel || !grid || !workspace) {
    return;
  }

  if (activeTool === "merge" && mergeModule) {
    mergeModule.resetPDFMerge();
  }

  activeTool = null;

  panel.querySelectorAll("[data-pdf-tool]").forEach((button) => {
    button.classList.remove("selected");
  });

  workspace.innerHTML = "";
  workspace.hidden = true;

  grid.hidden = false;

  panel.scrollIntoView({
    behavior: "smooth",
    block: "start",
  });
}

function renderWorkspaceLoading(workspace) {
  workspace.innerHTML = `
    <div class="pdf-workspace-loading">
      <span class="small-spinner"></span>

      <span>
        Werkzeug wird geladen …
      </span>
    </div>
  `;
}

function renderWorkspaceError(workspace, message) {
  workspace.innerHTML = `
    ${renderBackButton()}

    <div class="pdf-tool-placeholder">
      <strong>
        Werkzeug konnte nicht geladen werden
      </strong>

      <span>
        ${escapeHTML(message)}
      </span>
    </div>
  `;

  setupBackButton();
}

function renderComingSoon(workspace, tool) {
  workspace.innerHTML = `
    ${renderBackButton()}

    <div class="pdf-tool-placeholder">
      <strong>
        ${escapeHTML(getPDFToolName(tool))}
      </strong>

      <span>
        Dieses Werkzeug bauen wir als Nächstes.
      </span>
    </div>
  `;

  setupBackButton();
}

function renderBackButton() {
  return `
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
  `;
}

function setupBackButton() {
  const button = document.getElementById("pdf-tool-back");

  button?.addEventListener("click", closePDFTool);
}

function getPDFToolName(tool) {
  const names = {
    merge: "PDF zusammenfügen",

    split: "PDF trennen",

    compress: "PDF komprimieren",

    edit: "PDF bearbeiten",

    encrypt: "PDF verschlüsseln",

    decrypt: "Passwort entfernen",

    rotate: "Seiten drehen",

    delete: "Seiten löschen",

    extract: "Seiten extrahieren",

    sort: "Seiten sortieren",

    web: "Webseite in PDF",

    optimize: "PDF optimieren",

    redact: "PDF schwärzen",

    create: "PDF erstellen",
  };

  return names[tool] ?? "PDF Werkzeug";
}

function escapeHTML(value) {
  const element = document.createElement("div");

  element.textContent = value;

  return element.innerHTML;
}
