import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { ThemeProvider as PmmThemeProvider } from './themes/theme.provider';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <PmmThemeProvider>
      <App />
    </PmmThemeProvider>
  </React.StrictMode>
);

