import { useContext } from 'react';
import { NavigationContext } from './navigation.context';

export const useNavigation = () => useContext(NavigationContext);
