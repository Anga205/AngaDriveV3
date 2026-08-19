function normalizeRoute(route: string): string {
  return route ? (route.startsWith("/") ? route : `/${route}`) : "/";
}

export function apiUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);

  if (!import.meta.env.DEV) {
    const assetsUrl =
      (window as Window & { ASSETS_URL?: string }).ASSETS_URL?.trim();

    return assetsUrl
      ? `${assetsUrl.replace(/\/$/, "")}${normalizedRoute}`
      : normalizedRoute;
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