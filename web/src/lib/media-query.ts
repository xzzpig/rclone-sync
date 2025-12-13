import { createSignal, onCleanup, onMount } from 'solid-js';

export function useMediaQuery(query: string) {
  const [matches, setMatches] = createSignal(false);

  onMount(() => {
    const media = window.matchMedia(query);
    setMatches(media.matches);

    const listener = () => setMatches(media.matches);
    media.addEventListener('change', listener);

    onCleanup(() => media.removeEventListener('change', listener));
  });

  return matches;
}

export const useIsMobile = () => useMediaQuery('(max-width: 768px)');
export const useIsDesktop = () => useMediaQuery('(min-width: 769px)');
