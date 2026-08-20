export const DYNAMIC_ICON_IMPORT_MAP = {
  'pmm-titled': () => import('icons/pmm-titled.svg?react'),
  'pmm-titled-outlined': () => import('icons/pmm-titled-outlined.svg?react'),
  'pmm-rounded': () => import('icons/pmm-rounded.svg?react'),
  'status-at-risk': () => import('icons/status-at-risk.svg?react'),
  'status-down': () => import('icons/status-down.svg?react'),
  'status-updating': () => import('icons/status-updating.svg?react'),
  // todo: move to peak-ui
  'emergency-home': () => import('icons/emergency-home.svg?react'),
  // todo: move to peak-ui
  'chat-info-outlined': () => import('icons/chat-info-outlined.svg?react'),
};

export const VIEWBOX_MAP: Partial<
  Record<keyof typeof DYNAMIC_ICON_IMPORT_MAP, string>
> = {
  'pmm-rounded': '0 0 160 160',
  'pmm-titled': '0 0 141 48',
  'pmm-titled-outlined': '0 0 252 113',
};
