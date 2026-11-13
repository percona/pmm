// Content widths. The paper's right border sits outside them, so the drawer
// takes up DRAWER_BORDER_WIDTH more.
export const DRAWER_WIDTH = 256;

export const DRAWER_CLOSED_WIDTH = 48;

export const DRAWER_BORDER_WIDTH = 1;

// The space the drawer takes up, its border included.
export const drawerWidth = (open: boolean) =>
  (open ? DRAWER_WIDTH : DRAWER_CLOSED_WIDTH) + DRAWER_BORDER_WIDTH;

// Open and close durations in ms; everything in the sidebar moves with them.
export const DRAWER_OPEN_DURATION = 250;

export const DRAWER_CLOSE_DURATION = 200;
