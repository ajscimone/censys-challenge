CREATE TYPE access_level AS ENUM ('PRIVATE', 'ORGANIZATION', 'SHARED');

CREATE TABLE collections(
    id SERIAL PRIMARY KEY,
    uid UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    name TEXT NOT NULL, 
    data JSONB NOT NULL DEFAULT '{}', -- arbitrary data for the purpose of the challenge
    access_level access_level NOT NULL DEFAULT 'PRIVATE',
    owner_id INTEGER REFERENCES users(id) ON DELETE SET NULL, -- ive chose to allow for either an owner or an organization, need to ensure in code that it has one or the other. This is the more complex option than just having a user own it and deleting on cascade
    organization_id INTEGER REFERENCES organizations(id) ON DELETE SET NULL, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_collections_owner_id ON collections(owner_id);
CREATE INDEX idx_collections_organization_id ON collections(organization_id);
