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
