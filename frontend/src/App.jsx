import { useState } from 'react'
import AuthForm from './components/AuthForm'
import Chat from './pages/Chat'
import './App.css'

function App() {
  const [token, setToken] = useState(() => localStorage.getItem('token'))

  function handleLogin(token) {
    localStorage.setItem('token', token)
    setToken(token)
  }

  function handleLogout() {
    localStorage.removeItem('token')
    setToken(null)
  }

  if (!token) {
    return <AuthForm onLogin={handleLogin} />;
  }

  return <Chat token={token} onLogout={handleLogout} />;
}

export default App
