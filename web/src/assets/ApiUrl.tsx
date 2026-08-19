function normalizeRoute(route: string): string {
  return route ? (route.startsWith("/") ? route : `/${route}`) : "/";
}

export function apiUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);

  if (!import.meta.env.DEV) {
    const host = import.meta.env.VITE_ASSETS_URL?.trim();

    if (!host) return normalizedRoute;

    const baseUrl = /^https?:\/\//.test(host) ? host : `https://${host}`;

    return `${baseUrl.replace(/\/$/, "")}${normalizedRoute}`;
  }

  const host = import.meta.env.VITE_DEV_API_URL?.trim() || "localhost:8080";

  const isLocal =
    host.startsWith("127.0.0.1") ||
    host.startsWith("0.0.0.0") ||
    host.startsWith("localhost");

  return `${isLocal ? "http" : "https"}://${host}${normalizedRoute}`;
}

export function webSocketUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);

  if (!import.meta.env.DEV) {
    return `wss://${window.location.host}${normalizedRoute}`;
  }

  const host = import.meta.env.VITE_DEV_API_URL?.trim() || "localhost:8080";

  const isLocal =
    host.startsWith("127.0.0.1") ||
    host.startsWith("0.0.0.0") ||
    host.startsWith("localhost");

  return `${isLocal ? "ws" : "wss"}://${host}${normalizedRoute}`;
}