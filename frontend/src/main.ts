import { mount } from 'svelte';
import { initTheme, setThemeMode } from '@kenn-io/kit-ui';
import '@kenn-io/kit-ui/theme.css';
import './theme.css';
import App from './App.svelte';
import { migrateThemeKey, themeStorageKey } from './lib/theme';
import './app.css';
import './components.css';

try {
 migrateThemeKey(localStorage);
} catch {
 // Storage can be unavailable; the theme falls back to dark below.
}
initTheme({ storageKey: themeStorageKey });
// Default to charcoal while respecting explicit light and system preferences.
try {
 if (!localStorage.getItem(themeStorageKey)) setThemeMode('dark');
} catch {
 setThemeMode('dark');
}
mount(App, { target: document.getElementById('app')! });
