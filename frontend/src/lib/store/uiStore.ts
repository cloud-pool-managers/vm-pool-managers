import { writable } from 'svelte/store';

function persistedBool(key: string, defaultVal: boolean) {
  let initial = defaultVal;
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(key);
    if (saved !== null) initial = saved === 'true';
  }
  const store = writable<boolean>(initial);
  store.subscribe((val) => {
    if (typeof window !== 'undefined') localStorage.setItem(key, String(val));
  });
  return store;
}

function darkModeDefault(): boolean {
  if (typeof window === 'undefined') return false;
  const saved = localStorage.getItem('ui_dark_mode');
  if (saved !== null) return saved === 'true';
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

export const simpleMode = persistedBool('ui_simple_mode', true);

export const darkMode = (() => {
  const initial = darkModeDefault();
  const store = writable<boolean>(initial);
  store.subscribe((val) => {
    if (typeof window !== 'undefined') localStorage.setItem('ui_dark_mode', String(val));
  });
  return store;
})();
