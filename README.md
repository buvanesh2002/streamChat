# streamChat

streamChat is a simple Go-based server that enables real-time message streaming over HTTP using Server-Sent Events (SSE). It supports joining/leaving chat rooms, sending messages, and streaming live updates to connected clients.

Features:

=> Join/Leave chat with a unique client ID

=> Broadcast messages to all connected clients

=> Real-time message streaming using Server-Sent Events (SSE)

=> Automatic disconnection after 2 minutes of inactivity

Endpoints:

POST /join?id=your_unique_id

POST /send?id=your_id&message=your_message

POST /message?id=your_id

POST /leave?id=your_id


Example:

curl http://localhost:7020/join?id=1254






