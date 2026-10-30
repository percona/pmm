import { PointerEvent, RefObject, useRef } from 'react';
import { flushSync } from 'react-dom';

// How far the pointer travels before a press becomes a drag, so a click on a
// row or its handle stays a click.
const DRAG_THRESHOLD = 4;

interface DragState {
  pointerId: number;
  startY: number;
  from: number;
  to: number;
  rows: HTMLElement[];
  rects: DOMRect[];
  dragging: boolean;
}

type RowBox = Pick<DOMRect, 'top' | 'height'>;

// Where a dragged row is and which place it takes. Dragged as far as it goes,
// its centre reaches the end row's centre, and a row gives up its place once
// the dragged centre reaches its own, the same way in both directions, so the
// first and the last place are always reachable.
export const findDragTarget = (
  rows: RowBox[],
  from: number,
  delta: number
): { offset: number; to: number } => {
  const centres = rows.map((row) => row.top + row.height / 2);
  const own = rows[from];
  const first = rows[0];
  const last = rows[rows.length - 1];
  const offset = Math.min(
    Math.max(delta, Math.min(first.top - own.top, centres[0] - centres[from])),
    Math.max(
      last.top + last.height - (own.top + own.height),
      centres[centres.length - 1] - centres[from]
    )
  );
  const centre = centres[from] + offset;
  const to = centres.reduce(
    (target, rowCentre, index) =>
      index > from && centre >= rowCentre
        ? Math.max(target, index)
        : index < from && centre <= rowCentre
          ? Math.min(target, index)
          : target,
    from
  );

  return { offset, to };
};

interface Options {
  listRef: RefObject<HTMLElement | null>;
  ids: string[];
  enabled: boolean;
  onMove: (from: number, to: number) => void;
}

// Pointer reordering for a short list whose rows carry `data-item-id`. While
// dragging, rows only move with transforms; the list re-renders once, on drop.
export const useSortableDrag = ({ listRef, ids, enabled, onMove }: Options) => {
  const drag = useRef<DragState | null>(null);

  const findRows = () =>
    ids.flatMap(
      (id) =>
        listRef.current?.querySelector<HTMLElement>(
          `[data-item-id="${CSS.escape(id)}"]`
        ) ?? []
    );

  const onPointerDown = (event: PointerEvent<HTMLElement>) => {
    if (!enabled || event.button !== 0) {
      return;
    }

    const row = (event.target as HTMLElement).closest<HTMLElement>(
      '[data-item-id]'
    );
    const rows = findRows();
    const from = row ? rows.indexOf(row) : -1;

    if (from === -1) {
      return;
    }

    drag.current = {
      pointerId: event.pointerId,
      startY: event.clientY,
      from,
      to: from,
      rows,
      rects: rows.map((item) => item.getBoundingClientRect()),
      dragging: false,
    };
  };

  const onPointerMove = (event: PointerEvent<HTMLElement>) => {
    const state = drag.current;

    if (!state || event.pointerId !== state.pointerId) {
      return;
    }

    const { rows, rects, from } = state;
    const delta = event.clientY - state.startY;

    if (!state.dragging) {
      if (Math.abs(delta) < DRAG_THRESHOLD) {
        return;
      }

      state.dragging = true;
      rows[from].dataset.dragging = 'true';

      // Keeps the drag going when the pointer leaves the list. It can fail if
      // the pointer is already gone, which only ends the drag sooner.
      try {
        event.currentTarget.setPointerCapture(event.pointerId);
      } catch {
        // Nothing to capture.
      }
    }

    const own = rects[from];
    const { offset, to } = findDragTarget(rects, from, delta);

    rows.forEach((row, index) => {
      const shift =
        index === from
          ? offset
          : from < index && index <= to
            ? -own.height
            : to <= index && index < from
              ? own.height
              : 0;

      row.style.transform = shift ? `translateY(${shift}px)` : '';
    });

    state.to = to;
  };

  const finish = (commit: boolean) => {
    const state = drag.current;

    drag.current = null;

    if (!state?.dragging) {
      return;
    }

    const { rows, from, to } = state;
    const dragged = rows[from];

    if (commit && to !== from) {
      // Reorder for real, then let the dropped row glide from where it was
      // let go into its new place.
      const releasedAt = dragged.getBoundingClientRect().top;

      rows.forEach((row) => {
        row.style.transition = 'none';
        row.style.transform = '';
      });
      flushSync(() => onMove(from, to));
      dragged.style.transform = `translateY(${
        releasedAt - dragged.getBoundingClientRect().top
      }px)`;
      delete dragged.dataset.dragging;
      dragged.getBoundingClientRect();
      rows.forEach((row) => {
        row.style.transition = '';
      });
      dragged.style.transform = '';
    } else {
      delete dragged.dataset.dragging;
      rows.forEach((row) => {
        row.style.transform = '';
      });
    }
  };

  return {
    onPointerDown,
    onPointerMove,
    onPointerUp: () => finish(true),
    onPointerCancel: () => finish(false),
    cancel: () => finish(false),
    isDragging: () => !!drag.current?.dragging,
  };
};
