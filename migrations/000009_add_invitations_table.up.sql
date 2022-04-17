CREATE TABLE IF NOT EXISTS invitation
(
    user_id        int REFERENCES users (id) ON DELETE CASCADE,
    publication_id int REFERENCES publication (id) ON DELETE CASCADE,
    CONSTRAINT invitation_pk
        PRIMARY KEY (user_id, publication_id)
);
