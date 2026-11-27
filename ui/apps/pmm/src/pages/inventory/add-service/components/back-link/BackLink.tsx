import ArrowBackOutlinedIcon from '@mui/icons-material/ArrowBackOutlined';
import Button from '@mui/material/Button';
import type { FC } from 'react';
import { Link } from 'react-router-dom';
import type { BackLinkProps } from './BackLink.types';

export const BackLink: FC<BackLinkProps> = ({ label, to, onClick }) => (
  <Button
    size="small"
    startIcon={<ArrowBackOutlinedIcon />}
    data-testid="add-service-back"
    sx={{ alignSelf: 'flex-start', color: 'text.primary' }}
    {...(onClick ? { onClick } : { component: Link, to })}
  >
    {label}
  </Button>
);
