import EditOffOutlined from '@mui/icons-material/EditOffOutlined';
import EditOutlined from '@mui/icons-material/EditOutlined';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import List from '@mui/material/List';
import { useTheme } from '@mui/material/styles';
import { Tooltip } from '@percona/peak-ui';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';
import { FC, KeyboardEvent, useEffect, useMemo, useRef, useState } from 'react';
import { NavItem } from 'types/navigation.types';
import SidebarNavItem from '../nav-item/SidebarNavItem';
import ReorderHandle from './ReorderHandle';
import { getStyles } from './StarredSection.styles';
import { moveItem, orderItems } from './StarredSection.utils';
import { useSortableDrag } from './useSortableDrag';

export const STARRED_ORDER_KEY = 'pmm-ui.sidebar.starred-order';

interface Props {
  heading: NavItem;
  items: NavItem[];
  activeItem: NavItem;
  drawerOpen: boolean;
  onItemClick?: () => void;
}

// The Starred heading and its entries, which can be put in any order: a quiet
// button on the heading turns the entries into rows to drag, and the same
// button turns them back into links.
const StarredSection: FC<Props> = ({
  heading,
  items,
  activeItem,
  drawerOpen,
  onItemClick,
}) => {
  const theme = useTheme();
  const styles = getStyles(theme);
  const [order, setOrder] = useLocalStorage<string[]>(STARRED_ORDER_KEY, []);
  const ordered = useMemo(() => orderItems(items, order), [items, order]);
  const [editing, setEditing] = useState(false);
  const [announcement, setAnnouncement] = useState('');
  const listRef = useRef<HTMLUListElement>(null);
  const toggleRef = useRef<HTMLButtonElement>(null);
  const instructionsId = `${heading.id}-reorder-help`;
  const handleId = (item: NavItem) => `${item.id}-reorder`;

  const move = (from: number, to: number) => {
    const next = moveItem(ordered, from, to);

    setOrder(next.map((item) => item.id));
    setAnnouncement(
      `${ordered[from].text} moved to position ${to + 1} of ${next.length}.`
    );
  };

  const drag = useSortableDrag({
    listRef,
    ids: ordered.map((item) => item.id),
    enabled: editing,
    onMove: move,
  });

  const stopEditing = () => {
    drag.cancel();
    setEditing(false);
    setAnnouncement('');
  };

  useEffect(() => {
    if (!drawerOpen) {
      setEditing(false);
    }
  }, [drawerOpen]);

  const handleKeyDown = (event: KeyboardEvent) => {
    if (!editing || event.key !== 'Escape') {
      return;
    }

    event.stopPropagation();

    if (drag.isDragging()) {
      drag.cancel();
    } else {
      stopEditing();
      // A handle may have had focus, and the handles are going away.
      toggleRef.current?.focus();
    }
  };

  const moveFromHandle = (item: NavItem, from: number, to: number) => {
    move(from, to);
    // The entry's row moves in the list, so give its handle focus again.
    requestAnimationFrame(() =>
      document.getElementById(handleId(item))?.focus()
    );
  };

  const toggleLabel = editing ? 'Done editing order' : 'Edit order';

  const toggle = (
    <Tooltip title={toggleLabel} placement="right" arrow>
      <IconButton
        ref={toggleRef}
        size="small"
        color={editing ? 'primary' : 'default'}
        aria-label={toggleLabel}
        onClick={editing ? stopEditing : () => setEditing(true)}
        data-reveal={editing ? undefined : true}
        data-testid="starred-edit-toggle"
      >
        {editing ? (
          <EditOffOutlined fontSize="small" />
        ) : (
          <EditOutlined fontSize="small" />
        )}
      </IconButton>
    </Tooltip>
  );

  return (
    <Box
      component="li"
      sx={[styles.root, editing && styles.editing]}
      onKeyDown={handleKeyDown}
      data-testid="starred-section"
    >
      <List
        ref={listRef}
        disablePadding
        aria-label={heading.text}
        onPointerDown={drag.onPointerDown}
        onPointerMove={drag.onPointerMove}
        onPointerUp={drag.onPointerUp}
        onPointerCancel={drag.onPointerCancel}
        onDragStart={(event) => editing && event.preventDefault()}
      >
        <SidebarNavItem
          item={heading}
          activeItem={activeItem}
          drawerOpen={drawerOpen}
          headingAccessory={drawerOpen && toggle}
        />
        {ordered.map((item, index) => (
          <SidebarNavItem
            key={item.id}
            item={item}
            activeItem={activeItem}
            drawerOpen={drawerOpen}
            onClick={onItemClick}
            disableLink={editing}
            trailingControl={
              editing && (
                <ReorderHandle
                  id={handleId(item)}
                  text={item.text ?? ''}
                  index={index}
                  count={ordered.length}
                  describedBy={instructionsId}
                  onMove={(to) => moveFromHandle(item, index, to)}
                  testId={`navitem-${item.id}-handle`}
                />
              )
            }
          />
        ))}
      </List>
      <Box id={instructionsId} sx={styles.visuallyHidden}>
        Drag to move, or use the arrow keys. Press Enter for more options.
      </Box>
      <Box aria-live="polite" sx={styles.visuallyHidden}>
        {announcement}
      </Box>
    </Box>
  );
};

export default StarredSection;
