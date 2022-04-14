CREATE TABLE IF NOT EXISTS article
(
    id             bigserial PRIMARY KEY,
    title          varchar(255)                NOT NULL,
    content        text                        NOT NULL,
    publication_id int                         NOT NULL REFERENCES publication (id) ON DELETE CASCADE,
    writer_id      int                         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at     timestamp(0) with time zone NOT NULL DEFAULT now(),
    version        int                         NOT NULL DEFAULT 1
);
