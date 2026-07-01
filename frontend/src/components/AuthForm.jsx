import { useState } from "react"

function AuthForm({ onLogin }) {
    const [isLogin, setIsLogin] = useState(true);
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    async function handleSubmit(e) {
        e.preventDefault();

        try {
            const res = await fetch(`http://localhost:8081/api/${isLogin ? 'login' : 'register'}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });
            const data = await res.json();

            if (!res.ok) {
                setError('Invalid username or password')
                return
            }

            onLogin(data.token);
        } catch {
            setError('Invalid username or password');
        }
    }

    return (
        <div className="min-h-screen bg-gray-100 flex items-center justify-center">
            <div className="bg-white rounded-2xl shadow-md w-full max-w-sm p-8">
                <h2 className="text-2xl font-semibold text-gray-800 mb-6 text-center">
                    {isLogin ? 'Welcome back' : 'Create an account'}
                </h2>

                <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                    <input
                        type="text"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        placeholder="Username"
                        className="border border-gray-300 rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                    <input
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder="Password"
                        className="border border-gray-300 rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />

                    {error && (
                        <p className="text-red-500 text-sm">{error}</p>
                    )}

                    <button
                        type="submit"
                        className="bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg py-2 text-sm transition-colors"
                    >
                        {isLogin ? 'Log in' : 'Register'}
                    </button>
                </form>

                <p
                    onClick={() => setIsLogin(!isLogin)}
                    className="mt-5 text-center text-sm text-gray-500 hover:text-blue-600 cursor-pointer"
                >
                    {isLogin ? "Don't have an account? Register" : 'Already have an account? Log in'}
                </p>
            </div>
        </div>
    )
}

export default AuthForm
