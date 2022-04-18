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
	Writer        *User
	URL           string
	CreatedAt     time.Time
	Version       int
}

type Like struct {
	Count    int
	HasLiked bool
}

func (a *Article) SetURL() {
	a.URL = slug.Make(a.Title) + "-" + strconv.FormatInt(a.ID, 10)
}

func (a *Article) Matches(url string) bool {
	return url == slug.Make(a.Title)
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

func (m *ArticleModel) Get(articleID int) (*Article, error) {
	query := `
		SELECT a.id, a.title, a.content, a.publication_id, a.writer_id, a.created_at, a.version
		FROM article a
		WHERE a.id = $1`

	a := &Article{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, query, articleID)
	err := row.Scan(&a.ID, &a.Title, &a.Content, &a.PublicationID, &a.WriterID, &a.CreatedAt, &a.Version)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}
	a.SetURL()

	return a, nil
}

func (m *ArticleModel) GetArticlesOfPublication(publication *Publication) ([]*Article, error) {
	query := `
		SELECT id, title, content, publication_id, writer_id, created_at, version
		FROM article
		WHERE publication_id = $1
		ORDER BY created_at DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var articles []*Article

	rows, err := m.DB.QueryContext(ctx, query, publication.ID)
	for rows.Next() {
		a := &Article{}
		err = rows.Scan(&a.ID, &a.Title, &a.Content, &a.PublicationID, &a.WriterID, &a.CreatedAt, &a.Version)
		if err != nil {
			return nil, err
		}
		a.SetURL()
		articles = append(articles, a)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (m *ArticleModel) GetNewestArticles(filters Filters) ([]*Article, Metadata, error) {
	query := `
		SELECT count(*) OVER(), id, title, content, publication_id, writer_id, created_at, version
		FROM article
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, filters.limit(), filters.offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	var articles []*Article

	for rows.Next() {
		a := &Article{}
		err = rows.Scan(&totalRecords, &a.ID, &a.Title, &a.Content, &a.PublicationID, &a.WriterID, &a.CreatedAt, &a.Version)
		if err != nil {
			return nil, Metadata{}, err
		}
		a.SetURL()

		articles = append(articles, a)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metaData := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return articles, metaData, nil
}

func (m *ArticleModel) LikesMany(articles []*Article, user *User) (map[int]*Like, error) {
	likes := make(map[int]*Like)

	for _, article := range articles {
		if _, ok := likes[int(article.ID)]; ok {
			continue
		}

		like, err := m.Likes(article, user)

		if err != nil {
			return nil, err
		}
		likes[int(article.ID)] = like
	}

	return likes, nil
}

func (m *ArticleModel) Likes(article *Article, user *User) (*Like, error) {
	query := `
		SELECT COUNT(*) as likes
		FROM article_like
		WHERE article_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, query, article.ID)

	like := &Like{}
	err := row.Scan(&like.Count)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	like.HasLiked, err = m.UserHasLiked(article, user)
	if err != nil {
		return nil, err
	}

	return like, nil
}

func (m *ArticleModel) UserHasLiked(article *Article, user *User) (bool, error) {
	if user == nil || article == nil {
		return false, nil
	}

	query := `
		SELECT 1
		FROM article_like al
		WHERE al.user_id = $1 AND al.article_id = $2
		LIMIT 1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var exists int
	err := m.DB.QueryRowContext(ctx, query, user.ID, article.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return exists == 1, nil
}
