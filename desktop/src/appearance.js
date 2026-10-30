export const editorThemes = {
  auto: 'Match app',
  vs: 'Light',
  'vs-dark': 'Dark',
  'hc-black': 'High contrast dark',
  'hc-light': 'High contrast light',
};
export function resolveAppearance(choice, systemDark) {
  return choice === 'light' || choice === 'dark' ? choice : systemDark ? 'dark' : 'light';
}
export function resolveEditorTheme(choice, appearance) {
  return choice !== 'auto' && Object.hasOwn(editorThemes, choice) ? choice : appearance === 'light' ? 'vs' : 'vs-dark';
}
export function readPreference(key, allowed, fallback) {
  try { const value = localStorage.getItem(key); return allowed.includes(value) ? value : fallback; }
  catch { return fallback; }
}
export function savePreference(key, value) { try { localStorage.setItem(key, value); } catch { /* Session preferences still work with unavailable storage. */ } }
