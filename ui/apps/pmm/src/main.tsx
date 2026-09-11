import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.tsx';

// Attaches the response interceptors to the axios instances. Imported for that side effect
// alone, and here rather than deeper in the tree so it runs before the first request.
import 'api/api.interceptors';

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

import '@fontsource/poppins/400.css';
import '@fontsource/poppins/500.css';
import '@fontsource/poppins/600.css';

import '@fontsource/roboto-mono';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
