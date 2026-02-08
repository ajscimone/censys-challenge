CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    uid UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
