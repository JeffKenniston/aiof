/**
 * AIOF Frontend Environment Configuration
 * Provides extremely robust, dynamic resolution of backend endpoints.
 * Prioritizes explicit environment variables, falling back to dynamic
 * hostname resolution to seamlessly support both local dev, LAN testing,
 * and production deployments.
 */

export const config = {
  get backendUrl(): string {
    if (import.meta.env.VITE_BACKEND_URL) {
      return import.meta.env.VITE_BACKEND_URL;
    }
    // Return empty string to make all fetches relative, passing through Vite's proxy
    return '';
  },

  get backendWsUrl(): string {
    if (import.meta.env.VITE_BACKEND_WS_URL) {
      return import.meta.env.VITE_BACKEND_WS_URL;
    }
    if (typeof window !== 'undefined') {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      return `${protocol}//${window.location.host}`;
    }
    return '';
  }
};

export const BACKEND_URL = config.backendUrl;
export const BACKEND_WS_URL = config.backendWsUrl;
