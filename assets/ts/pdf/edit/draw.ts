import {
  clamp,
  createDrawStroke,
  editorState,
  getPageDrawStrokes,
  type PDFDrawPoint,
  type PDFDrawStroke,
} from "./state.ts";

import { updatePDFEditSubmitState } from "./submit.ts";

import { updatePageChangeIndicators } from "./text.ts";

const minDrawWidth = 1;
const maxDrawWidth = 20;
const maxPointsPerStroke = 5000;
const minimumPointDistance = 0.001;

let drawModeActive = false;
let activePointerID: number | null = null;
let activeStrokeID: string | null = null;

export function setupPDFEditDraw(): boolean {
  const root = editorState.root;

  if (!root) {
    return false;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-edit-draw-toggle");

  const canvas = root.querySelector<HTMLElement>("#pdf-edit-canvas");

  const color = root.querySelector<HTMLInputElement>("#pdf-edit-draw-color");

  const width = root.querySelector<HTMLInputElement>("#pdf-edit-draw-width");

  const undo = root.querySelector<HTMLButtonElement>("#pdf-edit-draw-undo");

  const clear = root.querySelector<HTMLButtonElement>("#pdf-edit-draw-clear");

  if (!button || !canvas || !color || !width || !undo || !clear) {
    return false;
  }

  button.addEventListener("click", () => {
    setDrawMode(!drawModeActive);
  });

  /*
   * Sobald ein anderes Werkzeug gewählt wird,
   * verlassen wir den Zeichenmodus.
   */
  root
    .querySelector<HTMLButtonElement>("#pdf-edit-add-text")
    ?.addEventListener("click", () => {
      setDrawMode(false);
    });

  root
    .querySelector<HTMLButtonElement>("#pdf-edit-add-image")
    ?.addEventListener("click", () => {
      setDrawMode(false);
    });

  undo.addEventListener("click", undoDrawStroke);

  clear.addEventListener("click", clearDrawStrokes);

  canvas.addEventListener("pointerdown", beginDrawing, true);

  canvas.addEventListener("pointermove", continueDrawing, true);

  canvas.addEventListener("pointerup", finishDrawing, true);

  canvas.addEventListener("pointercancel", finishDrawing, true);

  updateDrawControls();

  return true;
}

export function renderDrawStrokes(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const svg = root.querySelector<SVGSVGElement>("#pdf-edit-draw-overlay");

  const preview = root.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

  if (!svg || !preview) {
    return;
  }

  const rectangle = preview.getBoundingClientRect();

  if (rectangle.width <= 0 || rectangle.height <= 0) {
    svg.replaceChildren();

    return;
  }

  svg.setAttribute("viewBox", `0 0 ${rectangle.width} ${rectangle.height}`);

  svg.replaceChildren();

  const strokes = getPageDrawStrokes(editorState.activePage);

  for (const stroke of strokes) {
    svg.append(createStrokePath(stroke, rectangle.width, rectangle.height));
  }

  updateDrawControls();
}

export function resetPDFEditDraw(): void {
  drawModeActive = false;
  activePointerID = null;
  activeStrokeID = null;

  updateDrawModeUI();
  updateDrawControls();
}

function setDrawMode(active: boolean): void {
  if (active && !editorState.activeUpload) {
    return;
  }

  drawModeActive = active;

  activePointerID = null;
  activeStrokeID = null;

  if (active) {
    editorState.selectedTextID = null;

    editorState.selectedImageID = null;

    hideObjectProperties();
  }

  updateDrawModeUI();
  updateDrawControls();
}

function updateDrawModeUI(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-edit-draw-toggle");

  const properties = root.querySelector<HTMLElement>(
    "#pdf-edit-draw-properties",
  );

  const canvas = root.querySelector<HTMLElement>("#pdf-edit-canvas");

  button?.classList.toggle("active", drawModeActive);

  button?.setAttribute("aria-pressed", drawModeActive ? "true" : "false");

  if (properties) {
    properties.hidden = !drawModeActive;
  }

  canvas?.classList.toggle("drawing", drawModeActive);
}

function beginDrawing(event: PointerEvent): void {
  if (!drawModeActive || !editorState.activeUpload || event.button !== 0) {
    return;
  }

  const canvas = event.currentTarget;

  if (!(canvas instanceof HTMLElement)) {
    return;
  }

  event.preventDefault();
  event.stopPropagation();

  const point = pointerPosition(event, canvas);

  const color = currentDrawColor();

  const width = currentDrawWidth();

  const stroke = createDrawStroke(color, width, point);

  const strokes = getPageDrawStrokes(editorState.activePage);

  strokes.push(stroke);

  editorState.pageDrawStrokes.set(editorState.activePage, strokes);

  activePointerID = event.pointerId;

  activeStrokeID = stroke.id;

  canvas.setPointerCapture(event.pointerId);

  renderDrawStrokes();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function continueDrawing(event: PointerEvent): void {
  if (
    !drawModeActive ||
    activePointerID !== event.pointerId ||
    !activeStrokeID
  ) {
    return;
  }

  const canvas = event.currentTarget;

  if (!(canvas instanceof HTMLElement)) {
    return;
  }

  event.preventDefault();
  event.stopPropagation();

  const stroke = getPageDrawStrokes(editorState.activePage).find(
    (candidate) => candidate.id === activeStrokeID,
  );

  if (!stroke || stroke.points.length >= maxPointsPerStroke) {
    return;
  }

  const point = pointerPosition(event, canvas);

  const previous = stroke.points[stroke.points.length - 1];

  if (previous && pointDistance(previous, point) < minimumPointDistance) {
    return;
  }

  stroke.points.push(point);

  renderDrawStrokes();
}

function finishDrawing(event: PointerEvent): void {
  if (activePointerID !== event.pointerId) {
    return;
  }

  const canvas = event.currentTarget;

  if (
    canvas instanceof HTMLElement &&
    canvas.hasPointerCapture(event.pointerId)
  ) {
    canvas.releasePointerCapture(event.pointerId);
  }

  activePointerID = null;
  activeStrokeID = null;

  updateDrawControls();
}

function pointerPosition(
  event: PointerEvent,
  canvas: HTMLElement,
): PDFDrawPoint {
  const rectangle = canvas.getBoundingClientRect();

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

function createStrokePath(
  stroke: PDFDrawStroke,
  width: number,
  height: number,
): SVGPathElement {
  const path = document.createElementNS("http://www.w3.org/2000/svg", "path");

  path.classList.add("pdf-edit-draw-stroke");

  path.setAttribute("d", buildStrokePath(stroke, width, height));

  path.setAttribute("fill", "none");

  path.setAttribute("stroke", stroke.color);

  path.setAttribute("stroke-width", String(previewStrokeWidth(stroke.width)));

  path.setAttribute("stroke-linecap", "round");

  path.setAttribute("stroke-linejoin", "round");

  return path;
}

function buildStrokePath(
  stroke: PDFDrawStroke,
  width: number,
  height: number,
): string {
  if (stroke.points.length === 0) {
    return "";
  }

  const first = stroke.points[0];

  if (!first) {
    return "";
  }

  let result = `M ${first.x * width} ${first.y * height}`;

  if (stroke.points.length === 1) {
    result += ` L ${first.x * width + 0.01} ${first.y * height}`;

    return result;
  }

  for (let index = 1; index < stroke.points.length; index++) {
    const point = stroke.points[index];

    if (!point) {
      continue;
    }

    result += ` L ${point.x * width} ${point.y * height}`;
  }

  return result;
}

function previewStrokeWidth(points: number): number {
  const root = editorState.root;

  const preview = root?.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

  if (!preview) {
    return points;
  }

  const naturalWidth = preview.naturalWidth;

  const displayedWidth = preview.getBoundingClientRect().width;

  /*
   * PDF-Vorschau wird mit 90 DPI erzeugt.
   * PDF-Punkte basieren auf 72 pt/in.
   */
  const pixelsPerPoint = 90 / 72;

  if (naturalWidth <= 0 || displayedWidth <= 0) {
    return points * pixelsPerPoint;
  }

  return points * pixelsPerPoint * (displayedWidth / naturalWidth);
}

function currentDrawColor(): string {
  const input = editorState.root?.querySelector<HTMLInputElement>(
    "#pdf-edit-draw-color",
  );

  return input?.value || "#dc2626";
}

function currentDrawWidth(): number {
  const input = editorState.root?.querySelector<HTMLInputElement>(
    "#pdf-edit-draw-width",
  );

  const value = Number(input?.value ?? 3);

  if (!Number.isFinite(value)) {
    return 3;
  }

  return clamp(value, minDrawWidth, maxDrawWidth);
}

function undoDrawStroke(): void {
  const strokes = getPageDrawStrokes(editorState.activePage);

  if (strokes.length === 0) {
    return;
  }

  strokes.pop();

  editorState.pageDrawStrokes.set(editorState.activePage, strokes);

  renderDrawStrokes();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function clearDrawStrokes(): void {
  const strokes = getPageDrawStrokes(editorState.activePage);

  if (strokes.length === 0) {
    return;
  }

  editorState.pageDrawStrokes.delete(editorState.activePage);

  renderDrawStrokes();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function updateDrawControls(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const undo = root.querySelector<HTMLButtonElement>("#pdf-edit-draw-undo");

  const clear = root.querySelector<HTMLButtonElement>("#pdf-edit-draw-clear");

  const hasStrokes = getPageDrawStrokes(editorState.activePage).length > 0;

  if (undo) {
    undo.disabled = !hasStrokes;
  }

  if (clear) {
    clear.disabled = !hasStrokes;
  }
}

function hideObjectProperties(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const text = root.querySelector<HTMLElement>("#pdf-edit-text-properties");

  const image = root.querySelector<HTMLElement>("#pdf-edit-image-properties");

  if (text) {
    text.hidden = true;
  }

  if (image) {
    image.hidden = true;
  }
}

function pointDistance(first: PDFDrawPoint, second: PDFDrawPoint): number {
  return Math.hypot(second.x - first.x, second.y - first.y);
}
