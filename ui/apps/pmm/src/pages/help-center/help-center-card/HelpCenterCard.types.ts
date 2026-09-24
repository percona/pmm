import type { SemanticTokens } from '@percona/peak-ui';

export type HelpCardChartKey = keyof SemanticTokens['charts'];

export interface HelpCardButton {
  text: string;
  target?: string;
  url?: string;
  to?: string;
  startIconName?: string;
  onClick?: () => void;
  dataTestId?: string;
}

export interface HelpCard {
  id: string;
  title: string;
  description: string;
  buttons: HelpCardButton[];
  adminOnly: boolean;
  borderColorKey?: HelpCardChartKey;
}

export interface HelpCenterCardProps {
  card: HelpCard;
}
