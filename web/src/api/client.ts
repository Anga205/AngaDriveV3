/**
 * Low-level websocket client helpers.
 *
 * All websocket traffic should go through `send` / `request` so that message
 * framing, error detection, and timeouts live in one place. Components should
 * not call `socket.send(JSON.stringify(...))` directly.
 */

export interface WsResult<T = any> {
  ok: boolean;
  data: T;
}

/** Send a fire-and-forget message. */
export function send(ws: WebSocket | undefined, type: string, data: unknown): boolean {
  if (!ws || ws.readyState !== WebSocket.OPEN) return false;
  ws.send(JSON.stringify({ type, data }));
  return true;
}

/**
 * Send a request and resolve with the first response of `responseType`.
 * Error payloads (a string, or an object with `error`) resolve with `ok: false`.
 */
export function request<T = any>(
  ws: WebSocket,
  type: string,
  data: unknown,
  responseType: string,
  timeoutMs = 15000,
): Promise<WsResult<T>> {
  return new Promise<WsResult<T>>((resolve) => {
    let settled = false;
    const finish = (result: WsResult<T>) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      ws.removeEventListener("message", onMessage);
      ws.removeEventListener("close", onClose);
      resolve(result);
    };
    const onMessage = (event: MessageEvent) => {
      let message: any;
      try {
        message = JSON.parse(event.data);
      } catch {
        return;
      }
      if (message.type !== responseType) return;
      const payload = message.data;
      const isError =
        typeof payload === "string" ||
        (payload && typeof payload === "object" && "error" in payload);
      finish({ ok: !isError, data: payload });
    };
    const onClose = () => finish({ ok: false, data: "connection closed" as T });
    const timer = setTimeout(() => finish({ ok: false, data: "timeout" as T }), timeoutMs);

    ws.addEventListener("message", onMessage);
    ws.addEventListener("close", onClose);
    ws.send(JSON.stringify({ type, data }));
  });
}
