import { User } from 'types/user.types';

export interface UserContextProps {
  isLoading: boolean;
  user?: User;
}
