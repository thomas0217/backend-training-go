CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );