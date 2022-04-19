CREATE TABLE IF NOT EXISTS comment
(
    id           bigserial PRIMARY KEY,
    created_at   timestamp(0) with time zone NOT NULL DEFAULT now(),
    commenter_id int                         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    article_id   int                         NOT NULL REFERENCES article (id) ON DELETE CASCADE,
    content      text                        NOT NULL,
    version      integer                     NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS comment_like
(
    user_id    int REFERENCES users (id) ON DELETE CASCADE,
    comment_id int REFERENCES comment (id) ON DELETE CASCADE,
    CONSTRAINT comment_like_pk
        PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX IF NOT EXISTS comment_commenter_id_idx ON comment (commenter_id);
CREATE INDEX IF NOT EXISTS comment_article_id_idx ON comment (article_id);
