import { useSyncExternalStore } from 'react';

export type Breakpoint = 'condensed' | 'semi-full' | 'full' | 'expanded';

const SM_QUERY = '(min-width: 640px)';
const MD_QUERY = '(min-width: 1024px)';
const LG_QUERY = '(min-width: 1440px)';

function getBreakpoint(): Breakpoint {
  if (typeof window === 'undefined') return 'full';
  if (window.matchMedia(LG_QUERY).matches) return 'expanded';
  if (window.matchMedia(MD_QUERY).matches) return 'full';
  if (window.matchMedia(SM_QUERY).matches) return 'semi-full';
  return 'condensed';
}

function subscribe(callback: () => void): () => void {
  if (typeof window === 'undefined') return () => {};
  const smMql = window.matchMedia(SM_QUERY);
  const mdMql = window.matchMedia(MD_QUERY);
  const lgMql = window.matchMedia(LG_QUERY);
  
  smMql.addEventListener('change', callback);
  mdMql.addEventListener('change', callback);
  lgMql.addEventListener('change', callback);
  
  return () => {
    smMql.removeEventListener('change', callback);
    mdMql.removeEventListener('change', callback);
    lgMql.removeEventListener('change', callback);
  };
}

export function useBreakpoint(): Breakpoint {
  return useSyncExternalStore(subscribe, getBreakpoint, () => 'full' as Breakpoint);
}
