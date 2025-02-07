import { Message } from 'messenger';

export type LinkVariablesMessage = Message<{
  id: string;
  url: string;
}>;

export type NavigateToMessage = Message<{
  to: string;
}>;
