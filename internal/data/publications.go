package data

import (
	"context"
	"database/sql"
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

func (m *PublicationModel) GetUsersPublications(userID int) (*Publications, error) {
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
