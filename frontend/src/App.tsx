import { Router, Route } from "@solidjs/router";
import HomePage from "./pages/HomePage";
import MyDrive from "./pages/MyDrive";
import { WebSocketProvider } from "./Websockets";
import MyCollections from "./pages/MyCollections";
import Account from "./pages/Account";

const App = () => {
  return (
    <WebSocketProvider>
      <Router>
        <Route path="/" component={HomePage} />
        <Route path="/my_drive" component={MyDrive}/>
        <Route path="/my_collections" component={MyCollections}/>
        <Route path="/account" component={Account}/>
      </Router>
    </WebSocketProvider>
  )
}

export default App
