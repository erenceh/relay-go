CREATE TABLE messages (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sender_id  UUID NOT NULL REFERENCES users(id),
    room_id    UUID NOT NULL REFERENCES rooms(id),
    body       TEXT NOT NULL,
    delivered  BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);