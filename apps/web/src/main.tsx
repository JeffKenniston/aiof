window.addEventListener('error', (e) => {
  document.body.innerHTML = '<div style="background:red;color:white;padding:20px;z-index:9999;position:absolute;top:0;left:0;width:100%;height:100%;"><h1>GLOBAL ERROR</h1><pre>' + (e.error?.stack || e.message) + '</pre></div>';
});
window.addEventListener('unhandledrejection', (e) => {
  document.body.innerHTML = '<div style="background:red;color:white;padding:20px;z-index:9999;position:absolute;top:0;left:0;width:100%;height:100%;"><h1>PROMISE ERROR</h1><pre>' + (e.reason?.stack || e.reason) + '</pre></div>';
});

import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.tsx';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
