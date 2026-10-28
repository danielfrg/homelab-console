import { mount } from 'svelte';
import { initTheme, setThemeMode } from '@kenn-io/kit-ui';
import '@kenn-io/kit-ui/theme.css';
import './theme.css';
import App from './App.svelte';
import './app.css';
import './components.css';

initTheme({ storageKey: 'homelab-launcher-theme' });
// Default to charcoal while respecting explicit light and system preferences.
try {
 if (!localStorage.getItem('homelab-launcher-theme')) setThemeMode('dark');
} catch {
 setThemeMode('dark');
}
mount(App, { target: document.getElementById('app')! });
