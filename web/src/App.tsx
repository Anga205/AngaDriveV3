import { Router, Route } from "@solidjs/router";
import HomePage from "./pages/HomePage";
import MyDrive from "./pages/MyDrive";


const App = () => {
  return (
    <Router>
      <Route path="/" component={HomePage} />
      <Route path="/my_drive" component={MyDrive}/>
    </Router>
  )
}

export default App
