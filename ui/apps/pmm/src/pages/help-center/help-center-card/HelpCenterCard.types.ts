export interface HelpCardButton {
  text: string;
  target?: string;
  url?: string;
  to?: string;
  startIconName?: string;
  onClick?: () => void;
}

export interface HelpCard {
  id: string;
  title: string;
  description: string;
  buttons: HelpCardButton[];
  adminOnly: boolean;
  borderColor?: string;
}

export interface HelpCenterCardProps {
  card: HelpCard;
}
