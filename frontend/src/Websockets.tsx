import { createContext, onCleanup, useContext, createSignal, ParentComponent } from 'solid-js';
import { SocketStatus } from './types/types';


type WebSocketContextType = {
  socket: WebSocket;
  status: () => SocketStatus;
};

const WebSocketContext = createContext<WebSocketContextType>();

const WebSocketProvider: ParentComponent = (props) => {
  const [status, setStatus] = createSignal<SocketStatus>("connecting");
  
  const createSocket = () => {
    const host = window.location.host;
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    const isViteDev = import.meta.env.DEV;
    const wsUrl = isViteDev ? "ws://localhost:8080/ws" : `${protocol}://${host}/ws`;
    
    const ws = new WebSocket(wsUrl);
    setStatus("connecting");
    
    ws.onopen = () => {
      setStatus("connected");
      console.log("WebSocket connected");
    };
    
    ws.onclose = () => {
      setStatus("disconnected");
      console.log("WebSocket disconnected");
    };
    
    ws.onerror = (error) => {
      setStatus("error");
      console.error("WebSocket error:", error);
    };
    
    return ws;
  };

  const wsInstance = createSocket();

  onCleanup(() => {
    if (wsInstance.readyState === WebSocket.OPEN) {
      wsInstance.close();
    }
  });

  return (
    <WebSocketContext.Provider value={{
      socket: wsInstance,
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