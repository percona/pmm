export interface HelpCardButton {
  text: string;
  target?: string;
  url?: string;
  startIconName?: string;
}

export interface HelpCardType {
  id: string;
  title: string;
  description: string;
  buttons: HelpCardButton[];
  adminOnly: boolean;
  borderColor?: string;
}

export interface HelpCenterCardProps {
  card: HelpCardType;
}
