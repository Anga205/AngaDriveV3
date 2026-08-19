function normalizeRoute(route: string): string {
  return route ? (route.startsWith("/") ? route : `/${route}`) : "/";
}

function apiHost(): string {
  return import.meta.env.VITE_API_URL?.trim() || "localhost:8080";
}

function isLocalHost(host: string): boolean {
  return (
    host.startsWith("127.0.0.1") ||
    host.startsWith("0.0.0.0") ||
    host.startsWith("localhost")
  );
}

export function apiUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);
  const host = apiHost();

  return `${isLocalHost(host) ? "http" : "https"}://${host}${normalizedRoute}`;
}

export function webSocketUrl(route: string): string {
  const normalizedRoute = normalizeRoute(route);

  if (!import.meta.env.DEV) {
    return `wss://${window.location.host}${normalizedRoute}`;
  }

  const host = apiHost();

  return `${isLocalHost(host) ? "ws" : "wss"}://${host}${normalizedRoute}`;
}