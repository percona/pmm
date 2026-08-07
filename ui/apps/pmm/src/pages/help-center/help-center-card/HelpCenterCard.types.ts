import type { SemanticTokens } from '@percona/percona-ui';

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
