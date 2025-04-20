import { Router, Route } from "@solidjs/router";
import HomePage from "./pages/HomePage";
import MyDrive from "./pages/MyDrive";
import { WebSocketProvider } from "./Websockets";

const App = () => {
  return (
    <WebSocketProvider>
      <Router>
        <Route path="/" component={HomePage} />
        <Route path="/my_drive" component={MyDrive}/>
      </Router>
    </WebSocketProvider>
  )
}

export default App
