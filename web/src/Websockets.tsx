import { createContext, onCleanup, useContext, createSignal, ParentComponent, Accessor } from 'solid-js';
import { SocketStatus } from './library/types';
import { fetchFilesAndCollections, generateClientToken } from './library/functions';
import { clearStoredCredentials, isPasswordMigrated, migrateCachedPassword, wsRequest } from './library/password';
import { webSocketUrl } from './assets/ApiUrl';

const RECONNECT_DELAY = 300;
const wsUrl = webSocketUrl("/ws");

type WebSocketContextType = {
  socket: Accessor<WebSocket | undefined>;
  status: () => SocketStatus;
};

const WebSocketContext = createContext<WebSocketContextType>();

const WebSocketProvider: ParentComponent = (props) => {
  const [status, setStatus] = createSignal<SocketStatus>("connecting");
  const [webSocketInstance, setWebSocketInstance] = createSignal<WebSocket>();
  
  let reconnectTimeoutId: number | undefined;

  const createAndConnectSocket = () => {
    if (reconnectTimeoutId) {
      clearTimeout(reconnectTimeoutId);
      reconnectTimeoutId = undefined;
    }

    // Clean up any existing socket instance
    const oldSocket = webSocketInstance();
    if (oldSocket) {
      // Nullify handlers to prevent them from firing on the old instance, especially onclose
      oldSocket.onopen = null;
      oldSocket.onmessage = null; 
      oldSocket.onerror = null;
      oldSocket.onclose = null; // Crucial to prevent old onclose from re-triggering logic
      if (oldSocket.readyState !== WebSocket.CLOSED && oldSocket.readyState !== WebSocket.CLOSING) {
        oldSocket.close();
      }
    }
    
    setStatus("connecting");

    const ws = new WebSocket(wsUrl);
    ws.onopen = async () => {
      setStatus("connected");
      if (reconnectTimeoutId) { // Clear any pending reconnect if we successfully connected
        clearTimeout(reconnectTimeoutId);
        reconnectTimeoutId = undefined;
      }
      const userEmail = localStorage.getItem("email") || "";
      const userPassword = localStorage.getItem("password") || "";
      if (userEmail || userPassword) {
        if (!(userEmail && userPassword)) {
          clearStoredCredentials();
          fetchFilesAndCollections(ws);
        } else {
          // A cached password without the scheme marker is plaintext from the
          // pre-migration frontend. Upgrade it in place before logging in.
          if (!isPasswordMigrated()) {
            const migrated = await migrateCachedPassword(ws, userEmail, userPassword);
            if (!migrated) {
              // The cached plaintext is stale; drop it and continue as a guest.
              clearStoredCredentials();
              fetchFilesAndCollections(ws);
              return;
            }
          }
          // localStorage now holds the SHA-256 digest.
          const result = await wsRequest(
            ws,
            "login",
            { email: userEmail, password: localStorage.getItem("password") || "" },
            "login_response",
          );
          if (!result.ok) {
            console.error("Login failed:", result.data);
            clearStoredCredentials();
            localStorage.setItem("token", generateClientToken());
          } else {
            localStorage.removeItem("token");
          }
          fetchFilesAndCollections(ws);
        }
      } else {
        fetchFilesAndCollections(ws);
      }
    };
    
    ws.onclose = () => {
      // Only attempt to reconnect if this instance was the one we intended to be active
      if (webSocketInstance() === ws || !webSocketInstance()) { // also reconnect if webSocketInstance was cleared
        setStatus("reconnecting");
        reconnectTimeoutId = setTimeout(createAndConnectSocket, RECONNECT_DELAY);
      }
    };
    
    ws.onerror = (error) => {
      if (webSocketInstance() === ws) {   // Only handle errors for the current instance
        setStatus("error");
        console.error("WebSocket error:", error);
      }
    };
    
    setWebSocketInstance(ws);
  };

  // Initial connection attempt
  createAndConnectSocket();

  onCleanup(() => {
    if (reconnectTimeoutId) {
      clearTimeout(reconnectTimeoutId);
    }
    const ws = webSocketInstance();
    if (ws) {
      // Prevent onclose from triggering reconnection during component cleanup
      ws.onopen = null;
      ws.onmessage = null;
      ws.onerror = null;
      ws.onclose = null; 
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close();
      }
    }
    setWebSocketInstance(undefined); // Clean up the signal
  });

  return (
    <WebSocketContext.Provider value={{
      socket: webSocketInstance, // Pass the accessor directly
      status
    }}>
      {props.children}
    </WebSocketContext.Provider>
  );
};

const useWebSocket = (): WebSocketContextType => {
  const context = useContext(WebSocketContext);
  if (!context) throw new Error('useWebSocket must be used within WebSocketProvider');
  return context;
};

export { WebSocketProvider, useWebSocket };
export type { SocketStatus }; // Export SocketStatus if it's defined here and not in types/types