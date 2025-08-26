export interface HelpCardButton {
  text: string;
  target?: string;
  url?: string;
  startIconName?: string;
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
