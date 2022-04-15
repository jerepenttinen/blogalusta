package data

import (
	"context"
	"database/sql"
	"github.com/gosimple/slug"
	"strconv"
	"time"
)

type Article struct {
	ID            int64
	Title         string
	Content       string
	PublicationID int64
	WriterID      int64
	URL           string
	CreatedAt     time.Time
	Version       int
}

func (a *Article) SetURL() {
	a.URL = slug.Make(a.Title) + "-" + strconv.FormatInt(a.ID, 10)
}

type ArticleModel struct {
	DB *sql.DB
}

func (m *ArticleModel) Publish(writer *User, publication *Publication, title, content string) (*Article, error) {
	query := `
		INSERT INTO article (title, content, publication_id, writer_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, title, content, publication_id, writer_id, created_at, version`

	a := &Article{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, query, title, content, publication.ID, writer.ID)
	err := row.Scan(&a.ID, &a.Title, &a.Content, &a.PublicationID, &a.WriterID, &a.CreatedAt, &a.Version)
	if err != nil {
		return nil, err
	}
	a.SetURL()

	return a, nil
}
