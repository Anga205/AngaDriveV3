import { Router, Route } from "@solidjs/router";
import HomePage from "./pages/HomePage";
import { MyDrive } from "./pages/MyDrive";
import { useWebSocket, WebSocketProvider } from "./Websockets";
import MyCollections from "./pages/MyCollections";
import Account from "./pages/Account";
import { AppContext, ContextProvider } from "./Context";
import { createEffect, useContext } from "solid-js";
import { UniversalMessageHandler } from "./library/functions";
import CollectionPage from "./pages/Collection";


const UncontextedApp = () => {
  const { socket: getSocket } = useWebSocket();
  const ctx = useContext(AppContext)!;
  createEffect(() => {
    const socket = getSocket();
    if (!socket) return;
    socket.addEventListener("message", (event) => {
      UniversalMessageHandler(event, ctx);
    });
  })
  return (
    <Router>
      <Route path="/" component={HomePage} />
      <Route path="/my_drive" component={MyDrive}/>
      <Route path="/my_collections" component={MyCollections}/>
      <Route path="/account" component={Account}/>
      <Route path="/collection/*" component={CollectionPage} />
    </Router>
  )
}

const App = () => {
  return (
    <WebSocketProvider>
      <ContextProvider>
        <UncontextedApp />
      </ContextProvider>
    </WebSocketProvider>
  )
}



export default App
