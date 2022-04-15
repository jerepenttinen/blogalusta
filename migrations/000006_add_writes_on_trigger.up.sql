CREATE OR REPLACE FUNCTION view_insert_publication()
    RETURNS TRIGGER AS
$BODY$
BEGIN
    INSERT INTO writes_on (user_id, publication_id)
    VALUES (NEW.owner_id, NEW.id);
    RETURN NULL;
END;
$BODY$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS publication_insert
    ON publication;

CREATE TRIGGER publication_insert
    AFTER INSERT
    ON publication
    FOR EACH ROW
EXECUTE FUNCTION view_insert_publication();