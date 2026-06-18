// Initialisation de l'i18n (svelte-i18n) avec dictionnaires synchrones :
// aucun chargement asynchrone, donc pas de waitLocale ni de souci SSR.
import { addMessages, init, locale } from 'svelte-i18n';
import { browser } from '$app/environment';
import fr from './locales/fr';
import en from './locales/en';

export const SUPPORTED_LOCALES = ['fr', 'en'] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];
export const DEFAULT_LOCALE: Locale = 'fr';

addMessages('fr', fr);
addMessages('en', en);

function initialLocale(): Locale {
  if (browser) {
    const saved = localStorage.getItem('ui_language');
    if (saved === 'fr' || saved === 'en') return saved;
  }
  return DEFAULT_LOCALE;
}

init({
  fallbackLocale: DEFAULT_LOCALE,
  initialLocale: initialLocale(),
});

// Bascule la langue active (la persistance est gérée par le store uiStore.language).
export function setLocale(lang: Locale) {
  locale.set(lang);
}
