CREATE TABLE IF NOT EXISTS publication
(
    id          bigserial PRIMARY KEY,
    name        text                        NOT NULL,
    url         text                        NOT NULL UNIQUE,
    description text                        NOT NULL,
    owner_id    int                         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  timestamp(0) with time zone NOT NULL DEFAULT now(),
    version     int                         NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS writes_on
(
    user_id        int REFERENCES users (id) ON DELETE CASCADE,
    publication_id int REFERENCES publication (id) ON DELETE CASCADE,
    CONSTRAINT writes_on_pk
        PRIMARY KEY (user_id, publication_id)
);

CREATE TABLE IF NOT EXISTS subscribes_to
(
    user_id        int REFERENCES users (id) ON DELETE CASCADE,
    publication_id int REFERENCES publication (id) ON DELETE CASCADE,
    CONSTRAINT subscribes_to_pk
        PRIMARY KEY (user_id, publication_id)
);
