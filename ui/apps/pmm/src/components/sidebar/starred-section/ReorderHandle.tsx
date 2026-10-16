import DragIndicator from '@mui/icons-material/DragIndicator';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { Tooltip } from '@percona/peak-ui';
import { FC, KeyboardEvent, useState } from 'react';

interface Props {
  id: string;
  text: string;
  index: number;
  count: number;
  describedBy: string;
  onMove: (to: number) => void;
  testId: string;
}

// The drag handle doubles as the way to reorder without dragging: arrow keys
// move the entry, and a click or Enter opens Move up and Move down.
const ReorderHandle: FC<Props> = ({
  id,
  text,
  index,
  count,
  describedBy,
  onMove,
  testId,
}) => {
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
  const [tooltipOpen, setTooltipOpen] = useState(false);
  const menuId = `${id}-menu`;

  const move = (to: number) => {
    setMenuAnchor(null);
    onMove(to);
  };

  const handleKeyDown = (event: KeyboardEvent) => {
    const step =
      event.key === 'ArrowUp' ? -1 : event.key === 'ArrowDown' ? 1 : 0;
    const to = index + step;

    if (!step) {
      return;
    }

    event.preventDefault();

    if (to >= 0 && to < count) {
      onMove(to);
    }
  };

  return (
    <>
      {/* Shut once the handle is pressed, so it never trails behind a dragged
          row, and while the menu it opens is showing. */}
      <Tooltip
        title="Drag or click to reorder"
        placement="right"
        arrow
        open={tooltipOpen && !menuAnchor}
        onOpen={() => setTooltipOpen(true)}
        onClose={() => setTooltipOpen(false)}
      >
        <IconButton
          id={id}
          size="small"
          color="secondary"
          aria-label={`Reorder ${text}`}
          aria-describedby={describedBy}
          aria-haspopup="menu"
          aria-expanded={!!menuAnchor}
          aria-controls={menuAnchor ? menuId : undefined}
          onClick={(event) => setMenuAnchor(event.currentTarget)}
          onKeyDown={handleKeyDown}
          onPointerDown={() => setTooltipOpen(false)}
          data-testid={testId}
          sx={{ cursor: 'grab' }}
        >
          <DragIndicator fontSize="small" />
        </IconButton>
      </Tooltip>
      <Menu
        id={menuId}
        anchorEl={menuAnchor}
        open={!!menuAnchor}
        onClose={() => setMenuAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <MenuItem disabled={index === 0} onClick={() => move(index - 1)}>
          Move up
        </MenuItem>
        <MenuItem
          disabled={index === count - 1}
          onClick={() => move(index + 1)}
        >
          Move down
        </MenuItem>
      </Menu>
    </>
  );
};

export default ReorderHandle;
