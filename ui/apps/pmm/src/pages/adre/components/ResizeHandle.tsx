import { Box } from '@mui/material';
import { FC, KeyboardEvent, MouseEvent, useEffect, useRef } from 'react';

const KEY_STEP = 16;

export interface ResizeHandleProps {
  /** The panel it sizes: the one on its left or on its right. */
  side: 'left' | 'right';
  width: number;
  min: number;
  max: number;
  onResize: (width: number) => void;
  label: string;
}

/** A divider that sizes the panel beside it by dragging or with the arrow keys. */
export const ResizeHandle: FC<ResizeHandleProps> = ({
  side,
  width,
  min,
  max,
  onResize,
  label,
}) => {
  const stopDrag = useRef<(() => void) | null>(null);
  useEffect(() => () => stopDrag.current?.(), []);
  const clamp = (w: number) => Math.min(max, Math.max(min, Math.round(w)));
  // Dragging toward the panel shrinks it.
  const sign = side === 'left' ? 1 : -1;

  const onMouseDown = (e: MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = width;
    const move = (ev: globalThis.MouseEvent) =>
      onResize(clamp(startWidth + sign * (ev.clientX - startX)));
    const stop = () => {
      document.removeEventListener('mousemove', move);
      document.removeEventListener('mouseup', stop);
      document.body.style.removeProperty('user-select');
      document.body.style.removeProperty('cursor');
      stopDrag.current = null;
    };
    document.addEventListener('mousemove', move);
    document.addEventListener('mouseup', stop);
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'col-resize';
    stopDrag.current = stop;
  };

  const onKeyDown = (e: KeyboardEvent) => {
    const next = {
      ArrowRight: width + sign * KEY_STEP,
      ArrowLeft: width - sign * KEY_STEP,
      Home: min,
      End: max,
    }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    onResize(clamp(next));
  };

  return (
    <Box
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      aria-valuenow={width}
      aria-valuemin={min}
      aria-valuemax={max}
      tabIndex={0}
      onMouseDown={onMouseDown}
      onKeyDown={onKeyDown}
      sx={{
        flex: '0 0 9px',
        mx: '-4px',
        zIndex: 1,
        cursor: 'col-resize',
        display: 'flex',
        justifyContent: 'center',
        outline: 'none',
        '&::after': {
          content: '""',
          width: '1px',
          bgcolor: 'rgba(255,255,255,0.06)',
          transition: 'background-color 120ms, width 120ms',
        },
        '&:hover::after, &:focus-visible::after, &:active::after': {
          width: '3px',
          bgcolor: 'primary.main',
        },
      }}
    />
  );
};
