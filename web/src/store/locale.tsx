import * as m from '@/paraglide/messages.js';
import {
  extractLocaleFromNavigator,
  isLocale,
  locales,
  overwriteGetLocale,
  getLocale as paraglideGetLocale,
  type Locale,
} from '@/paraglide/runtime';
import { ParentComponent, createContext, createSignal, onMount, useContext } from 'solid-js';

const LOCALE_STORAGE_KEY = 'locale';

interface LocaleState {
  locale: Locale;
  isDetected: boolean;
}

interface LocaleActions {
  setLocale: (locale: Locale) => void;
  detectLanguage: () => Locale;
}

const LocaleContext = createContext<[LocaleState, LocaleActions]>();

// Create a module-level signal for locale
// This signal will be read by Paraglide's getLocale() through overwriteGetLocale
const [localeSignal, setLocaleSignal] = createSignal<Locale>(paraglideGetLocale());

// Override Paraglide's getLocale to read from our signal
// This makes all message function calls (m.hello()) reactive in SolidJS components
// eslint-disable-next-line solid/reactivity
overwriteGetLocale(() => localeSignal());

/**
 * Detects the user's preferred language from browser settings.
 * Falls back to base locale ('en') if detection fails.
 *
 * Priority order:
 * 1. localStorage (user preference)
 * 2. navigator.languages (browser preference)
 * 3. Base locale ('en')
 */
function detectLanguage(): Locale {
  // 1. Check localStorage for saved preference (browser only)
  if (typeof window !== 'undefined') {
    try {
      const saved = window.localStorage.getItem(LOCALE_STORAGE_KEY);
      if (saved && isLocale(saved)) {
        return saved as Locale;
      }
    } catch {
      // localStorage may not be available
    }
  }

  // 2. Check browser language
  const browserLocale = extractLocaleFromNavigator();
  if (browserLocale) {
    return browserLocale;
  }

  // 3. Fallback to base locale
  return paraglideGetLocale();
}

export const LocaleProvider: ParentComponent = (props) => {
  const [isDetected, setIsDetected] = createSignal(false);

  const state = {
    get locale() {
      return localeSignal();
    },
    get isDetected() {
      return isDetected();
    },
  };

  const actions: LocaleActions = {
    setLocale: (newLocale: Locale) => {
      if (!isLocale(newLocale)) {
        console.error(`Invalid locale: ${newLocale}. Available: ${locales.join(', ')}`);
        return;
      }

      // Update the signal - this will trigger reactivity in all components
      setLocaleSignal(newLocale);

      // Persist to localStorage (browser only)
      if (typeof window !== 'undefined') {
        try {
          window.localStorage.setItem(LOCALE_STORAGE_KEY, newLocale);
        } catch (err) {
          console.warn('Failed to save locale to localStorage:', err);
        }
      }
    },

    detectLanguage,
  };

  // Detect language on mount (client-side only)
  onMount(() => {
    const detected = detectLanguage();

    // Update the signal
    setLocaleSignal(detected);

    setIsDetected(true);
  });

  return (
    <LocaleContext.Provider value={[state, actions]}> {props.children}</LocaleContext.Provider>
  );
};

export const useLocale = () => {
  const context = useContext(LocaleContext);
  if (!context) {
    throw new Error(m.error_hookMissingProvider({ hook: 'useLocale', provider: 'LocaleProvider' }));
  }
  return context;
};

/**
 * Get the current locale directly (for use outside components)
 * This returns the reactive signal value
 */
export const getLocale = () => localeSignal();

/**
 * Get available locales
 */
export { locales };

/**
 * Check if a string is a valid locale
 */
export { isLocale };
