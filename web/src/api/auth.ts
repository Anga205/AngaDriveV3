/**
 * Centralized credential access.
 *
 * Every request that needs authentication should use `getAuth()` instead of
 * reading localStorage directly. This keeps the storage keys in one place and
 * makes it trivial to change the credential scheme later.
 */

export interface Auth {
  token: string;
  email: string;
  password: string;
}

const TOKEN_KEY = "token";
const EMAIL_KEY = "email";
const PASSWORD_KEY = "password";
const DISPLAY_NAME_KEY = "display_name";

/** Read the current credentials from storage. */
export function getAuth(): Auth {
  return {
    token: localStorage.getItem(TOKEN_KEY) || "",
    email: localStorage.getItem(EMAIL_KEY) || "",
    password: localStorage.getItem(PASSWORD_KEY) || "",
  };
}

/** True when any credential is present. */
export function hasCredentials(): boolean {
  const auth = getAuth();
  return Boolean(auth.token || auth.email || auth.password);
}

/** Persist the account token. */
export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

/** Persist the account email. */
export function setEmail(email: string): void {
  localStorage.setItem(EMAIL_KEY, email);
}

/** Persist the display name. */
export function setDisplayName(name: string): void {
  localStorage.setItem(DISPLAY_NAME_KEY, name);
}

/** Read the cached display name. */
export function getDisplayName(): string {
  return localStorage.getItem(DISPLAY_NAME_KEY) || "";
}

/** Read the cached email. */
export function getEmail(): string {
  return localStorage.getItem(EMAIL_KEY) || "";
}

/** Read the cached token. */
export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || "";
}
