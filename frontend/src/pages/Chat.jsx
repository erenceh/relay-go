import { useState, useEffect, useRef } from 'react'

function writeMessage(ws, text) {
    const payload = new TextEncoder().encode(text)
    const buf = new ArrayBuffer(4 + payload.length)
    const view = new DataView(buf)
    view.setUint32(0, payload.length, false)
    new Uint8Array(buf).set(payload, 4)
    ws.send(buf)
}

function readMessage(data) {
    return new Promise((resolve) => {
        const reader = new FileReader()
        reader.onload = () => {
            const buf = reader.result
            const view = new DataView(buf)
            const length = view.getUint32(0, false)
            const text = new TextDecoder().decode(new Uint8Array(buf, 4, length))
            resolve(text)
        }
        reader.readAsArrayBuffer(data)
    })
}

function Chat({ token, onLogout }) {
    const [messages, setMessages] = useState([])
    const [input, setInput] = useState('')
    const ws = useRef(null)

    function sendMessage() {
        if (!input.trim()) return
        writeMessage(ws.current, input)
        setInput('')
    }

    useEffect(() => {
        ws.current = new WebSocket(`ws://localhost:8081/ws?token=${token}`)

        ws.current.onmessage = async (event) => {
            const text = await readMessage(event.data)
            setMessages(prev => [...prev, text])
        }

        ws.current.onclose = () => {
            if (event.code !== 1000) {
                onLogout()
            }
        }

        return () => {
            ws.current.close()
        }
    }, [])

    return (
        <div className="flex flex-col h-screen">
            <button
                onClick={onLogout}
                className="bg-red-500 hover:bg-red-600 text-white px-4 py-2 text-sm"
            >
                Log Out
            </button>
            <div className="flex-1 overflow-y-auto p-4 space-y-2">
                {messages.map((msg, i) => (
                    <div key={i} className="text-sm text-gray-800">{msg}</div>
                ))}
            </div>
            <div className="flex gap-2 p-4 border-t">
                <input
                    type="text"
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && sendMessage()}
                    placeholder="Type a message..."
                    className="flex-1 border rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <button
                    onClick={sendMessage}
                    className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm"
                >
                    Send
                </button>
            </div>
        </div>
    )
}

export default Chat