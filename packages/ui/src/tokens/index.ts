export const tokens = {
  colors: {
    background: 'hsl(224, 71%, 4%)',
    foreground: 'hsl(213, 31%, 91%)',
    primary: 'hsl(210, 100%, 52%)',
    primaryForeground: 'hsl(224, 71%, 4%)',
    secondary: 'hsl(215, 27.9%, 16.9%)',
    secondaryForeground: 'hsl(210, 40%, 98%)',
    accent: 'hsl(262, 83%, 58%)',
    accentForeground: 'hsl(210, 40%, 98%)',
    destructive: 'hsl(0, 84.2%, 60.2%)',
    border: 'hsl(215, 27.9%, 16.9%)',
    glass: 'rgba(15, 23, 42, 0.65)',
  },
  typography: {
    fontSans: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    fontMono: '"Fira Code", "JetBrains Mono", monospace',
  },
  glassmorphism: {
    backdropFilter: 'blur(12px) saturate(180%)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    boxShadow: '0 8px 32px 0 rgba(0, 0, 0, 0.37)',
  },
  animations: {
    fast: '150ms cubic-bezier(0.4, 0, 0.2, 1)',
    normal: '300ms cubic-bezier(0.4, 0, 0.2, 1)',
    spring: { type: 'spring', stiffness: 300, damping: 30 },
  }
};

export type DesignTokens = typeof tokens;
