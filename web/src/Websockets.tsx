import { createContext, onCleanup, useContext, createSignal, ParentComponent, Accessor } from 'solid-js';
import { SocketStatus } from './library/types';
import { fetchFilesAndCollections, generateClientToken } from './library/functions';

const RECONNECT_DELAY = 300;
const host = window.location.host;
const protocol = window.location.protocol === "https:" ? "wss" : "ws";
const isViteDev = import.meta.env.DEV;
const wsUrl = isViteDev ? "ws://localhost:8080/ws" : `${protocol}://${host}/ws`;

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
    ws.onopen = () => {
      setStatus("connected");
      console.log("Websockets.tsx: WebSocket connected");
      if (reconnectTimeoutId) { // Clear any pending reconnect if we successfully connected
        clearTimeout(reconnectTimeoutId);
        reconnectTimeoutId = undefined;
      }
      const userEmail = localStorage.getItem("email") || "";
      const userPassword = localStorage.getItem("password") || "";
      if (userEmail || userPassword) {
        if (!(userEmail && userPassword)) {
          localStorage.removeItem("email");
          localStorage.removeItem("password");
          localStorage.removeItem("display_name")
          fetchFilesAndCollections(ws);
        } else {
          ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            if (message.type === "login_response"){
              if (message.data === "invalid credentials") {
                console.error("Login failed:", message.data.error);
                localStorage.removeItem("email");
                localStorage.removeItem("password");
                localStorage.removeItem("display_name");
                localStorage.setItem("token", generateClientToken());
              } else {
                localStorage.removeItem("token");
              }
              fetchFilesAndCollections(ws);
              ws.onmessage = () => {} // Clear the message handler after login to prevent further processing
            }
          };
          ws.send(JSON.stringify({
            type: "login",
            data: {
              email: userEmail,
              password: userPassword
            }
          }))
        }
      } else {
        fetchFilesAndCollections(ws);
      }
    };
    
    ws.onclose = () => {
      // Only attempt to reconnect if this instance was the one we intended to be active
      if (webSocketInstance() === ws || !webSocketInstance()) { // also reconnect if webSocketInstance was cleared
        setStatus("reconnecting");
        console.log("Websockets.tsx: WebSocket disconnected. Attempting to reconnect...");
        reconnectTimeoutId = setTimeout(createAndConnectSocket, RECONNECT_DELAY);
      } else {
        console.log("An old WebSocket instance closed.");
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