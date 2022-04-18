CREATE TABLE IF NOT EXISTS article_like (
    user_id int REFERENCES users (id) ON DELETE CASCADE,
    article_id int REFERENCES article (id) ON DELETE CASCADE,
    CONSTRAINT article_like_pk
        PRIMARY KEY (user_id, article_id)
);