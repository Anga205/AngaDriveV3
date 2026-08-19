function normalizeRoute(route: string): string {
  return route ? (route.startsWith("/") ? route : `/${route}`) : "/";
}

function isLocalHost(host: string): boolean {
  return (
    host.startsWith("127.0.0.1") ||
    host.startsWith("0.0.0.0") ||
    host.startsWith("localhost")
  );
}

function resolveHost(envVar: string | undefined): string {
  return envVar?.trim() || "localhost:8080";
}

export function apiUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);

  // In production the frontend is served by the Go backend, so internal API
  // calls use relative routes (same origin as the frontend).
  if (!import.meta.env.DEV) {
    return normalizedRoute;
  }

  const host = resolveHost(import.meta.env.VITE_API_URL);
  return `${isLocalHost(host) ? "http" : "https"}://${host}${normalizedRoute}`;
}

export function assetsUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);
  const host = resolveHost(import.meta.env.VITE_ASSETS_URL);

  return `${isLocalHost(host) ? "http" : "https"}://${host}${normalizedRoute}`;
}

export function webSocketUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);

  if (!import.meta.env.DEV) {
    return `wss://${window.location.host}${normalizedRoute}`;
  }

  const host = resolveHost(import.meta.env.VITE_API_URL);

  return `${isLocalHost(host) ? "ws" : "wss"}://${host}${normalizedRoute}`;
}