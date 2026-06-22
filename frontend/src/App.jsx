import { useState } from 'react'
import './App.css'

function App() {
  const [token, setToken] = useState(null)

  if (!token) {
    return <LoginForm onLogin={setToken} />;
  }

  return <ChatPage token={token} />;
}

export default App
