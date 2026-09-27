import { destroyPDFDelete, setupPDFDelete } from "./delete/index.ts";

import { destroyPDFExtract, setupPDFExtract } from "./extract/index.ts";

import { destroyPDFMerge, setupPDFMerge } from "./merge/index.ts";

import { destroyPDFRotate, setupPDFRotate } from "./rotate/index.ts";

import { destroyPDFSort, setupPDFSort } from "./sort/index.ts";

type PDFTool = "merge" | "sort" | "delete" | "extract" | "rotate" | string;

let activeTool: PDFTool | null = null;

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

    if (initializeWorkspace(workspace)) {
      return;
    }

    console.error("PDF workspace could not be initialized.");
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

function initializeWorkspace(workspace: HTMLElement): boolean {
  const merge = workspace.querySelector<HTMLElement>(
    '[data-pdf-workspace="merge"]',
  );

  if (merge) {
    setupPDFMerge(merge);

    return true;
  }

  const sort = workspace.querySelector<HTMLElement>(
    '[data-pdf-workspace="sort"]',
  );

  if (sort) {
    setupPDFSort(sort);

    return true;
  }

  const deleteWorkspace = workspace.querySelector<HTMLElement>(
    '[data-pdf-workspace="delete"]',
  );

  if (deleteWorkspace) {
    setupPDFDelete(deleteWorkspace);

    return true;
  }

  const extract = workspace.querySelector<HTMLElement>(
    '[data-pdf-workspace="extract"]',
  );

  if (extract) {
    setupPDFExtract(extract);

    return true;
  }

  const rotate = workspace.querySelector<HTMLElement>(
    '[data-pdf-workspace="rotate"]',
  );

  if (rotate) {
    setupPDFRotate(rotate);

    return true;
  }

  const placeholder = workspace.querySelector<HTMLElement>(
    '[data-pdf-workspace="placeholder"]',
  );

  return placeholder !== null;
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
  switch (activeTool) {
    case "merge":
      await destroyPDFMerge();

      break;

    case "sort":
      await destroyPDFSort();

      break;

    case "delete":
      await destroyPDFDelete();

      break;

    case "extract":
      await destroyPDFExtract();

      break;

    case "rotate":
      await destroyPDFRotate();

      break;
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
