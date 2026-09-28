import { setupRedactionInteraction } from "./interaction.ts";

import {
  clearCurrentPage,
  deleteSelectedRedaction,
  updateControls,
} from "./render.ts";

import { setupRedactSubmit } from "./submit.ts";

import { destroyRedactUpload, setupRedactUpload } from "./upload.ts";

import { redactState } from "./state.ts";

export function setupPDFRedact(workspace: HTMLElement): void {
  redactState.root = workspace;

  if (
    !setupRedactUpload() ||
    !setupRedactionInteraction() ||
    !setupRedactSubmit()
  ) {
    return;
  }

  const deleteButton =
    workspace.querySelector<HTMLButtonElement>("#pdf-redact-delete");

  const clearButton = workspace.querySelector<HTMLButtonElement>(
    "#pdf-redact-clear-page",
  );

  if (!deleteButton || !clearButton) {
    return;
  }

  deleteButton.addEventListener("click", deleteSelectedRedaction);

  clearButton.addEventListener("click", clearCurrentPage);

  updateControls();
}

export async function destroyPDFRedact(): Promise<void> {
  await destroyRedactUpload();

  redactState.root = null;
}
