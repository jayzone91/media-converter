import {
  clamp,
  createRedaction,
  findRedaction,
  getPageRedactions,
  redactState,
  type PDFRedaction,
} from "./state.ts";

import {
  removeRedaction,
  renderRedactions,
  updateControls,
  updatePageIndicators,
} from "./render.ts";

type InteractionMode = "create" | "move" | "resize";

interface RedactionInteraction {
  mode: InteractionMode;

  pointerId: number;

  id: string;

  startX: number;
  startY: number;

  originalX: number;
  originalY: number;

  originalWidth: number;
  originalHeight: number;
}

const minimumRedactionSize = 0.005;

let interaction: RedactionInteraction | null = null;

export function setupRedactionInteraction(): boolean {
  const overlay = redactOverlay();

  if (!overlay) {
    return false;
  }

  overlay.addEventListener("pointerdown", handlePointerDown);

  overlay.addEventListener("pointermove", handlePointerMove);

  overlay.addEventListener("pointerup", finishInteraction);

  overlay.addEventListener("pointercancel", finishInteraction);

  return true;
}

export function resetRedactionInteraction(): void {
  interaction = null;
}

function handlePointerDown(event: PointerEvent): void {
  if (event.button !== 0 || !redactState.activeUpload) {
    return;
  }

  const target = event.target;

  const overlay = event.currentTarget;

  if (!(target instanceof Element) || !(overlay instanceof HTMLElement)) {
    return;
  }

  const area = target.closest<HTMLElement>(".pdf-redact-area");

  if (!area) {
    beginCreateRedaction(event, overlay);

    return;
  }

  const id = area.dataset.redactionId;

  if (!id) {
    return;
  }

  const redaction = findRedaction(redactState.activePage, id);

  if (!redaction) {
    return;
  }

  if (target.closest(".pdf-redact-resize-handle")) {
    beginResizeRedaction(event, overlay, redaction);

    return;
  }

  beginMoveRedaction(event, overlay, redaction);
}

function beginCreateRedaction(event: PointerEvent, overlay: HTMLElement): void {
  event.preventDefault();

  const point = normalizedPointer(event, overlay);

  const redaction = createRedaction(point.x, point.y, 0, 0);

  const redactions = getPageRedactions(redactState.activePage);

  redactions.push(redaction);

  redactState.pageRedactions.set(redactState.activePage, redactions);

  redactState.selectedID = redaction.id;

  interaction = {
    mode: "create",

    pointerId: event.pointerId,

    id: redaction.id,

    startX: point.x,

    startY: point.y,

    originalX: point.x,

    originalY: point.y,

    originalWidth: 0,
    originalHeight: 0,
  };

  overlay.setPointerCapture(event.pointerId);

  renderRedactions();
  updateControls();
}

function beginMoveRedaction(
  event: PointerEvent,
  overlay: HTMLElement,
  redaction: PDFRedaction,
): void {
  event.preventDefault();
  event.stopPropagation();

  const point = normalizedPointer(event, overlay);

  redactState.selectedID = redaction.id;

  interaction = {
    mode: "move",

    pointerId: event.pointerId,

    id: redaction.id,

    startX: point.x,

    startY: point.y,

    originalX: redaction.x,

    originalY: redaction.y,

    originalWidth: redaction.width,

    originalHeight: redaction.height,
  };

  overlay.setPointerCapture(event.pointerId);

  renderRedactions();
  updateControls();
}

function beginResizeRedaction(
  event: PointerEvent,
  overlay: HTMLElement,
  redaction: PDFRedaction,
): void {
  event.preventDefault();
  event.stopPropagation();

  const point = normalizedPointer(event, overlay);

  redactState.selectedID = redaction.id;

  interaction = {
    mode: "resize",

    pointerId: event.pointerId,

    id: redaction.id,

    startX: point.x,

    startY: point.y,

    originalX: redaction.x,

    originalY: redaction.y,

    originalWidth: redaction.width,

    originalHeight: redaction.height,
  };

  overlay.setPointerCapture(event.pointerId);

  renderRedactions();
  updateControls();
}

function handlePointerMove(event: PointerEvent): void {
  if (!interaction || interaction.pointerId !== event.pointerId) {
    return;
  }

  const overlay = event.currentTarget;

  if (!(overlay instanceof HTMLElement)) {
    return;
  }

  const redaction = findRedaction(redactState.activePage, interaction.id);

  if (!redaction) {
    return;
  }

  event.preventDefault();

  const point = normalizedPointer(event, overlay);

  switch (interaction.mode) {
    case "create":
      updateCreatedRedaction(redaction, point.x, point.y);

      break;

    case "move":
      updateMovedRedaction(redaction, point.x, point.y);

      break;

    case "resize":
      updateResizedRedaction(redaction, point.x, point.y);

      break;
  }

  renderRedactions();
}

function updateCreatedRedaction(
  redaction: PDFRedaction,
  x: number,
  y: number,
): void {
  if (!interaction) {
    return;
  }

  const left = Math.min(interaction.startX, x);

  const top = Math.min(interaction.startY, y);

  const right = Math.max(interaction.startX, x);

  const bottom = Math.max(interaction.startY, y);

  redaction.x = left;
  redaction.y = top;

  redaction.width = right - left;

  redaction.height = bottom - top;
}

function updateMovedRedaction(
  redaction: PDFRedaction,
  x: number,
  y: number,
): void {
  if (!interaction) {
    return;
  }

  const deltaX = x - interaction.startX;

  const deltaY = y - interaction.startY;

  redaction.x = clamp(interaction.originalX + deltaX, 0, 1 - redaction.width);

  redaction.y = clamp(interaction.originalY + deltaY, 0, 1 - redaction.height);
}

function updateResizedRedaction(
  redaction: PDFRedaction,
  x: number,
  y: number,
): void {
  if (!interaction) {
    return;
  }

  const deltaX = x - interaction.startX;

  const deltaY = y - interaction.startY;

  redaction.width = clamp(
    interaction.originalWidth + deltaX,
    minimumRedactionSize,
    1 - redaction.x,
  );

  redaction.height = clamp(
    interaction.originalHeight + deltaY,
    minimumRedactionSize,
    1 - redaction.y,
  );
}

function finishInteraction(event: PointerEvent): void {
  if (!interaction || interaction.pointerId !== event.pointerId) {
    return;
  }

  const overlay = event.currentTarget;

  if (
    overlay instanceof HTMLElement &&
    overlay.hasPointerCapture(event.pointerId)
  ) {
    overlay.releasePointerCapture(event.pointerId);
  }

  const id = interaction.id;

  interaction = null;

  const redaction = findRedaction(redactState.activePage, id);

  if (
    redaction &&
    (redaction.width < minimumRedactionSize ||
      redaction.height < minimumRedactionSize)
  ) {
    removeRedaction(id);

    return;
  }

  renderRedactions();
  updateControls();
  updatePageIndicators();
}

function normalizedPointer(
  event: PointerEvent,
  overlay: HTMLElement,
): {
  x: number;
  y: number;
} {
  const rectangle = overlay.getBoundingClientRect();

  if (rectangle.width <= 0 || rectangle.height <= 0) {
    return {
      x: 0,
      y: 0,
    };
  }

  return {
    x: clamp((event.clientX - rectangle.left) / rectangle.width, 0, 1),

    y: clamp((event.clientY - rectangle.top) / rectangle.height, 0, 1),
  };
}

function redactOverlay(): HTMLElement | null {
  return (
    redactState.root?.querySelector<HTMLElement>("#pdf-redact-overlay") ?? null
  );
}
