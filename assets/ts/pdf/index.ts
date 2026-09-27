import { renderPDFMerge, resetPDFMerge } from "./merge.ts";

type PDFTool =
  | "merge"
  | "split"
  | "compress"
  | "edit"
  | "encrypt"
  | "decrypt"
  | "rotate"
  | "delete"
  | "extract"
  | "sort"
  | "web"
  | "optimize"
  | "redact"
  | "create";

let activeTool: PDFTool | null = null;

export function setupPDFUI(): void {
  const panel = document.querySelector<HTMLElement>('[data-panel="pdf"]');

  if (!panel) {
    return;
  }

  const buttons = panel.querySelectorAll<HTMLButtonElement>("[data-pdf-tool]");

  for (const button of buttons) {
    button.addEventListener("click", () => {
      const tool = parsePDFTool(button.dataset.pdfTool);

      if (!tool) {
        return;
      }

      openPDFTool(tool);
    });
  }
}

function openPDFTool(tool: PDFTool): void {
  const panel = document.querySelector<HTMLElement>('[data-panel="pdf"]');

  const grid = panel?.querySelector<HTMLElement>(".tool-grid");

  const workspace = document.querySelector<HTMLElement>("#pdf-tool-workspace");

  if (!panel || !grid || !workspace) {
    return;
  }

  activeTool = tool;

  panel
    .querySelectorAll<HTMLButtonElement>("[data-pdf-tool]")
    .forEach((button) => {
      button.classList.toggle("selected", button.dataset.pdfTool === tool);
    });

  grid.hidden = true;
  workspace.hidden = false;

  switch (tool) {
    case "merge":
      renderPDFMerge(workspace, closePDFTool);

      break;

    default:
      renderComingSoon(workspace, tool);
  }
}

function closePDFTool(): void {
  const panel = document.querySelector<HTMLElement>('[data-panel="pdf"]');

  const grid = panel?.querySelector<HTMLElement>(".tool-grid");

  const workspace = document.querySelector<HTMLElement>("#pdf-tool-workspace");

  if (!panel || !grid || !workspace) {
    return;
  }

  if (activeTool === "merge") {
    resetPDFMerge();
  }

  activeTool = null;

  panel
    .querySelectorAll<HTMLButtonElement>("[data-pdf-tool]")
    .forEach((button) => {
      button.classList.remove("selected");
    });

  workspace.replaceChildren();
  workspace.hidden = true;

  grid.hidden = false;

  panel.scrollIntoView({
    behavior: "smooth",
    block: "start",
  });
}

function renderComingSoon(workspace: HTMLElement, tool: PDFTool): void {
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

  const button = workspace.querySelector<HTMLButtonElement>("#pdf-tool-back");

  button?.addEventListener("click", closePDFTool);
}

function renderBackButton(): string {
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

function parsePDFTool(value: string | undefined): PDFTool | null {
  switch (value) {
    case "merge":
    case "split":
    case "compress":
    case "edit":
    case "encrypt":
    case "decrypt":
    case "rotate":
    case "delete":
    case "extract":
    case "sort":
    case "web":
    case "optimize":
    case "redact":
    case "create":
      return value;

    default:
      return null;
  }
}

function getPDFToolName(tool: PDFTool): string {
  const names: Record<PDFTool, string> = {
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

  return names[tool];
}

function escapeHTML(value: string): string {
  const element = document.createElement("div");

  element.textContent = value;

  return element.innerHTML;
}
