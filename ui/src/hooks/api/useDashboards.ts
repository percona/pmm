import { useMutation } from '@tanstack/react-query';
import { starDashboard, unstarDashboard } from 'api/dashboards';

export const useStarDashboard = () =>
  useMutation({
    mutationKey: ['star-dashboard'],
    mutationFn: (uid: string) => starDashboard(uid),
  });

export const useUnstarDashboard = () =>
  useMutation({
    mutationKey: ['unstar-dashboard'],
    mutationFn: (uid: string) => unstarDashboard(uid),
  });
