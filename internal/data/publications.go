package data

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gosimple/slug"
	"time"
)

type Publication struct {
	ID          int64
	Name        string
	URL         string
	Description string
	OwnerID     int64
	CreatedAt   time.Time
	Version     int
}

func (p *Publication) GetBaseURL() string {
	return fmt.Sprintf("/%s", p.URL)
}

func (p *Publication) GetSettingsURL() string {
	return fmt.Sprintf("/%s/settings", p.URL)
}

func (p *Publication) GetAboutURL() string {
	return fmt.Sprintf("/%s/about", p.URL)
}

func (p *Publication) GetArticleURL(article *Article) string {
	return fmt.Sprintf("/%s/%s", p.URL, article.URL)
}

type Publications struct {
	SubscribesTo []*Publication
	WritesOn     []*Publication
	Owns         []*Publication
}

type PublicationModel struct {
	DB *sql.DB
}

func (m *PublicationModel) GetBySlug(slug string) (*Publication, error) {
	query := `
		SELECT id, name, url, description, owner_id, created_at, version
		FROM publication
		WHERE url = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, query, slug)

	p := &Publication{}
	err := row.Scan(&p.ID, &p.Name, &p.URL, &p.Description, &p.OwnerID, &p.CreatedAt, &p.Version)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return p, nil
}

func (m *PublicationModel) GetUsersPublications(userID int64) (*Publications, error) {
	ps := &Publications{}

	qt := []struct {
		query string
		pubs  *[]*Publication
	}{
		{
			query: `
			SELECT
				wp.id,
				wp.name,
				wp.url,
				wp.description,
				wp.owner_id,
				wp.created_at,
				wp.version
			FROM
				users u
			JOIN writes_on wo on u.id = wo.user_id
			JOIN publication wp on wo.publication_id = wp.id
			WHERE u.id = $1;`,
			pubs: &ps.WritesOn,
		},
		{
			query: `
			SELECT
				sp.id,
				sp.name,
				sp.url,
				sp.description,
				sp.owner_id,
				sp.created_at,
				sp.version
			FROM
				users u
			JOIN subscribes_to st on u.id = st.user_id
			JOIN publication sp on st.publication_id = sp.id
			WHERE u.id = $1;`,
			pubs: &ps.SubscribesTo,
		},
		{
			query: `
			SELECT
				op.id,
				op.name,
				op.url,
				op.description,
				op.owner_id,
				op.created_at,
				op.version
			FROM
				users u
			JOIN publication op on u.id = op.owner_id
			WHERE u.id = $1;`,
			pubs: &ps.Owns,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	for _, q := range qt {
		rows, err := m.DB.QueryContext(ctx, q.query, userID)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			p := &Publication{}
			err = rows.Scan(&p.ID, &p.Name, &p.URL, &p.Description, &p.OwnerID, &p.CreatedAt, &p.Version)
			if err != nil {
				return nil, err
			}

			*q.pubs = append(*q.pubs, p)
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}
	}

	return ps, nil
}

func (m *PublicationModel) DeleteByID(userID, publicationID int64) error {
	query := `
		DELETE
		FROM publication p
		WHERE p.owner_id = $1 AND p.id = $2;`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID, publicationID)

	if err != nil {
		return err
	}

	return nil
}

func (m *PublicationModel) Insert(userID int64, name, description string) (string, error) {
	query := `
		INSERT INTO publication (name, url, description, owner_id)
		VALUES ($1, $2, $3, $4)`

	url := slug.Make(name)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, name, url, description, userID)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "publication_url_key"`:
			return "", ErrDuplicateRecord
		default:
			return "", err
		}
	}

	return url, nil
}

func (m *PublicationModel) UserIsWriter(publication *Publication, user *User) (bool, error) {
	if user == nil || publication == nil {
		return false, nil
	}

	query := `
		SELECT 1
		FROM writes_on wo
		WHERE wo.user_id = $1 AND wo.publication_id = $2
		LIMIT 1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var exists int
	err := m.DB.QueryRowContext(ctx, query, user.ID, publication.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func (m *PublicationModel) UserIsSubscribed(publication *Publication, user *User) (bool, error) {
	if user == nil || publication == nil {
		return false, nil
	}

	query := `
		SELECT 1
		FROM subscribes_to st
		WHERE st.user_id = $1 AND st.publication_id = $2
		LIMIT 1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var exists int
	err := m.DB.QueryRowContext(ctx, query, user.ID, publication.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func (m *PublicationModel) Invite(publication *Publication, user *User) error {
	query := `
		INSERT INTO invitation (user_id, publication_id)
		VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user.ID, publication.ID)
	if err.Error() == `pq: duplicate key value violates unique constraint "invitation_pk"` {
		return ErrDuplicateRecord
	} else if err != nil {
		return err
	}

	return nil
}

func (m *PublicationModel) Withdraw(publication *Publication, id int) error {
	query := `
		DELETE FROM invitation
		WHERE user_id = $1 AND publication_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id, publication.ID)
	if err == sql.ErrNoRows {
		return ErrRecordNotFound
	} else if err != nil {
		return err
	}

	return nil
}
