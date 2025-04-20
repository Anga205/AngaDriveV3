import { createContext, onCleanup, useContext } from 'solid-js';

type WebSocketContextType = WebSocket | undefined;
const WebSocketContext = createContext<WebSocketContextType>();

const WebSocketProvider = (props: any) => {
  const host = window.location.host;
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const isViteDev = import.meta.env.DEV;
  const socket = new WebSocket(isViteDev ? "ws://localhost:8080/ws" : `${protocol}://${host}/ws`);

  onCleanup(() => {
    if (socket.readyState === WebSocket.OPEN) {
      socket.close();
    }
  });

  return (
    <WebSocketContext.Provider value={socket}>
      {props.children}
    </WebSocketContext.Provider>
  );
}

const useWebSocket = () => {
  return useContext(WebSocketContext);
}

export { WebSocketProvider, useWebSocket }