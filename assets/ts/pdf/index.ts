import { destroyPDFMerge, setupPDFMerge } from "./merge.ts";

let activeTool: string | null = null;

export function setupPDFUI(): void {
  const panel = document.querySelector<HTMLElement>('[data-panel="pdf"]');

  const grid = panel?.querySelector<HTMLElement>(".tool-grid");

  const workspace = document.querySelector<HTMLElement>("#pdf-tool-workspace");

  if (!panel || !grid || !workspace) {
    return;
  }

  setupToolSelection(panel, grid, workspace);

  setupWorkspaceEvents(panel, grid, workspace);
}

function setupToolSelection(
  panel: HTMLElement,
  grid: HTMLElement,
  workspace: HTMLElement,
): void {
  const buttons = panel.querySelectorAll<HTMLButtonElement>("[data-pdf-tool]");

  for (const button of buttons) {
    button.addEventListener("click", () => {
      const tool = button.dataset.pdfTool;

      if (!tool) {
        return;
      }

      activeTool = tool;

      for (const candidate of buttons) {
        candidate.classList.toggle("selected", candidate === button);
      }

      grid.hidden = true;
      workspace.hidden = false;

      workspace.replaceChildren();

      const loading = document.createElement("div");

      loading.className = "pdf-workspace-loading";

      const spinner = document.createElement("span");

      spinner.className = "small-spinner";

      const text = document.createElement("span");

      text.textContent = "Werkzeug wird geladen …";

      loading.append(spinner, text);

      workspace.append(loading);
    });
  }
}

function setupWorkspaceEvents(
  panel: HTMLElement,
  grid: HTMLElement,
  workspace: HTMLElement,
): void {
  document.body.addEventListener("htmx:afterSwap", (event: Event) => {
    if (!isWorkspaceSwap(event, workspace)) {
      return;
    }

    const merge = workspace.querySelector<HTMLElement>(
      '[data-pdf-workspace="merge"]',
    );

    if (merge) {
      setupPDFMerge(merge);
    }
  });

  workspace.addEventListener("click", (event: MouseEvent) => {
    const target = event.target;

    if (!(target instanceof Element)) {
      return;
    }

    const back = target.closest<HTMLElement>("[data-pdf-back]");

    if (!back) {
      return;
    }

    void closePDFWorkspace(panel, grid, workspace);
  });
}

function isWorkspaceSwap(event: Event, workspace: HTMLElement): boolean {
  const customEvent = event as CustomEvent<{
    target?: Element;
  }>;

  return customEvent.detail?.target === workspace;
}

async function closePDFWorkspace(
  panel: HTMLElement,
  grid: HTMLElement,
  workspace: HTMLElement,
): Promise<void> {
  if (activeTool === "merge") {
    await destroyPDFMerge();
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
