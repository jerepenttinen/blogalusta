CREATE TABLE IF NOT EXISTS image
(
    id            bigserial PRIMARY KEY,
    image_data    bytea NOT NULL
);
