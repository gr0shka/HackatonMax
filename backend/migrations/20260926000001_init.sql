-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS postgis;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    interests JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_interests ON users USING GIN (interests);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_friends (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, friend_id),
    CONSTRAINT chk_user_friends_not_self CHECK (user_id <> friend_id)
);

CREATE INDEX IF NOT EXISTS idx_user_friends_friend_id ON user_friends(friend_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS places (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL DEFAULT '',
    category VARCHAR(100) NOT NULL DEFAULT '',
    rating NUMERIC(3, 2) NOT NULL DEFAULT 0.00,
    avg_duration_min INT NOT NULL DEFAULT 30,
    lat DOUBLE PRECISION NOT NULL,
    lon DOUBLE PRECISION NOT NULL,
    geom GEOMETRY(Point, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_places_geom ON places USING GIST (geom);
CREATE INDEX IF NOT EXISTS idx_places_geom_geog ON places USING GIST ((geom::geography));
CREATE INDEX IF NOT EXISTS idx_places_category ON places (category);
CREATE UNIQUE INDEX IF NOT EXISTS idx_places_external_id ON places (external_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_places_geom()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.lat <> OLD.lat OR NEW.lon <> OLD.lon THEN
            NEW.geom := ST_SetSRID(ST_MakePoint(NEW.lon, NEW.lat), 4326);
        END IF;
    ELSIF TG_OP = 'INSERT' THEN
        IF NEW.geom IS NULL AND NEW.lon IS NOT NULL AND NEW.lat IS NOT NULL THEN
            NEW.geom := ST_SetSRID(ST_MakePoint(NEW.lon, NEW.lat), 4326);
        ELSIF NEW.geom IS NOT NULL AND (NEW.lon IS NULL OR NEW.lat IS NULL) THEN
            NEW.lon := ST_X(NEW.geom);
            NEW.lat := ST_Y(NEW.geom);
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_places_geom ON places;
CREATE TRIGGER trg_places_geom
BEFORE INSERT OR UPDATE OF lat, lon ON places
FOR EACH ROW
EXECUTE FUNCTION set_places_geom();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_places_geom ON places;
DROP FUNCTION IF EXISTS set_places_geom();
DROP TABLE IF EXISTS places;
DROP TABLE IF EXISTS user_friends;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
