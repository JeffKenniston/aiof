import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [
    tailwindcss(),
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'AIOF Workstation Web PWA',
        short_name: 'AIOF Web',
        description: 'Agentic AI Orchestration Framework Web Workstation',
        theme_color: '#0f172a',
        background_color: '#090d16',
        display: 'standalone',
        icons: [
          {
            src: '/favicon.svg',
            sizes: '192x192',
            type: 'image/svg+xml'
          }
        ]
      }
    })
  ],
  server: {
    port: 5174,
    proxy: {
      '/v1': 'http://localhost:8080'
    }
  },
  optimizeDeps: {
    exclude: ['@aiof/ui', '@aiof/rxdb-store', '@aiof/contracts']
  }
});
