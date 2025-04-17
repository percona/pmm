export interface CardButton {
  text: string;
  target: string;
  url: string;
  startIconName: string;
}

export interface CardType {
  id: string;
  title: string;
  description: string;
  buttons: Array<CardButton>;
  adminOnly: boolean;
  borderColor?: string;
}

export interface HelpCenterCardProps {
  card: CardType;
  shouldDisplayCard: (adminOnly: boolean) => boolean;
}
