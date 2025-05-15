import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.tsx';
import { CrossFrameMessenger } from '@pmm/shared';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

console.log('CrossFrameMessenger - from PMM UI', CrossFrameMessenger);
