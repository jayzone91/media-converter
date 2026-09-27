export function setupDocumentDrag(
  item: HTMLElement,
  id: string,
  root: HTMLElement,
  reorder: (sourceID: string, targetID: string) => void,
): void {
  item.addEventListener("dragstart", (event: DragEvent) => {
    if (
      event.target instanceof Element &&
      event.target.closest(".pdf-page-strip")
    ) {
      event.preventDefault();

      return;
    }

    item.classList.add("dragging");

    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";

      event.dataTransfer.setData("text/plain", id);
    }
  });

  item.addEventListener("dragend", () => {
    item.classList.remove("dragging");

    clearDragTargets(root);
  });

  item.addEventListener("dragover", (event: DragEvent) => {
    event.preventDefault();

    const draggedID = event.dataTransfer?.getData("text/plain");

    if (!draggedID || draggedID === id) {
      return;
    }

    clearDragTargets(root);

    item.classList.add("drag-target");

    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
  });

  item.addEventListener("dragleave", () => {
    item.classList.remove("drag-target");
  });

  item.addEventListener("drop", (event: DragEvent) => {
    event.preventDefault();

    const draggedID = event.dataTransfer?.getData("text/plain");

    item.classList.remove("drag-target");

    if (!draggedID || draggedID === id) {
      return;
    }

    reorder(draggedID, id);
  });
}

function clearDragTargets(root: HTMLElement): void {
  root
    .querySelectorAll<HTMLElement>(".pdf-file-item.drag-target")
    .forEach((item) => {
      item.classList.remove("drag-target");
    });
}
