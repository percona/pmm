import { ThemeContext } from '@grafana/data';
import { config, getAppEvents, ThemeChangedEvent } from '@grafana/runtime';
import React, { FC, PropsWithChildren, useEffect, useState } from 'react';

export const ThemeProvider: FC<PropsWithChildren> = ({ children }) => {
  const [theme, setTheme] = useState(config.theme2);

  useEffect(() => {
    getAppEvents().subscribe(ThemeChangedEvent, (event) => {
      setTheme(event.payload);
    });
  }, []);

  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
};
