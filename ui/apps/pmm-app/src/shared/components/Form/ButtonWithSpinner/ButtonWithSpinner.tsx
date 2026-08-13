import type { FC } from 'react';
import React from 'react';
import { cx } from '@emotion/css';
import type { ButtonProps } from '@grafana/ui';
import { Button, Spinner } from '@grafana/ui';
import * as styles from './ButtonWithSpinner.styles';

type ButtonWithSpinnerProps = ButtonProps & {
  isLoading?: boolean;
};

export const ButtonWithSpinner: FC<ButtonWithSpinnerProps> = ({
  children,
  disabled,
  className = '',
  isLoading = false,
  ...props
}) => (
  <Button
    className={cx(styles.Button, className)}
    size="md"
    disabled={isLoading || disabled}
    {...props}
  >
    {isLoading ? <Spinner /> : children}
  </Button>
);
