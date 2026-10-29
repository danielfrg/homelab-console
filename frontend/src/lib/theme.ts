export const themeStorageKey = "homelab-console-theme";
const legacyThemeStorageKey = "homelab-launcher-theme";

type KeyValueStorage = Pick<Storage, "getItem" | "setItem" | "removeItem">;

// migrateThemeKey keeps the theme chosen under the app's old name.
export function migrateThemeKey(storage: KeyValueStorage): void {
  const legacy = storage.getItem(legacyThemeStorageKey);
  if (legacy === null) return;
  if (storage.getItem(themeStorageKey) === null) storage.setItem(themeStorageKey, legacy);
  storage.removeItem(legacyThemeStorageKey);
}
