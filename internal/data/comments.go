package data

import (
	"context"
	"database/sql"
	"time"
)

type CommentModel struct {
	DB *sql.DB
}

type Comment struct {
	ID          int
	CreatedAt   time.Time
	CommenterID int
	ArticleID   int
	Content     string
	Version     int
}

func (m *CommentModel) Get(commentID int) (*Comment, error) {
	query := `
		SELECT id, created_at, commenter_id, article_id, content, version
		FROM comment
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, query, commentID)

	c := &Comment{}

	err := row.Scan(&c.ID, &c.CreatedAt, &c.CommenterID, &c.ArticleID, &c.Content, &c.Version)
	if err != nil {
		return nil, err
	}

	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return c, nil
}

func (m *CommentModel) Count(article *Article) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM comment
		WHERE article_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int
	err := m.DB.QueryRowContext(ctx, query, article.ID).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return count, nil
}

func (m *CommentModel) Retrieve(article *Article) ([]*Comment, error) {
	query := `
		SELECT id, created_at, commenter_id, article_id, content, version, COUNT(cl.comment_id) as likes
		FROM comment
		LEFT JOIN comment_like cl on comment.id = cl.comment_id
		WHERE article_id = $1
		GROUP BY id
		ORDER BY likes DESC, id DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	comments := make([]*Comment, 0)
	rows, err := m.DB.QueryContext(ctx, query, article.ID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		c := &Comment{}
		var likes int

		err = rows.Scan(&c.ID, &c.CreatedAt, &c.CommenterID, &c.ArticleID, &c.Content, &c.Version, &likes)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return comments, nil
}

func (m *CommentModel) Commenters(comments []*Comment, userMap map[int]*User) (map[int]*User, error) {
	if userMap == nil {
		userMap = make(map[int]*User, 0)
	}

	query := `SELECT id, name, email, created_at, image_id FROM users WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	for _, comment := range comments {
		if _, ok := userMap[comment.CommenterID]; ok {
			continue
		}

		s := &User{}
		err := m.DB.QueryRowContext(ctx, query, comment.CommenterID).Scan(&s.ID, &s.Name, &s.Email, &s.CreatedAt, &s.ImageID)

		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		} else if err != nil {
			return nil, err
		}

		userMap[s.ID] = s
	}

	return userMap, nil
}

func (m *CommentModel) LikesMany(comments []*Comment, user *User) (map[int]*Like, error) {
	likes := make(map[int]*Like)

	for _, comment := range comments {
		if _, ok := likes[comment.ID]; ok {
			continue
		}

		like, err := m.Likes(comment, user)

		if err != nil {
			return nil, err
		}
		likes[comment.ID] = like
	}

	return likes, nil
}

func (m *CommentModel) Likes(comment *Comment, user *User) (*Like, error) {
	query := `
		SELECT COUNT(*) as likes
		FROM comment_like
		WHERE comment_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, query, comment.ID)

	like := &Like{}
	err := row.Scan(&like.Count)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	like.HasLiked, err = m.UserHasLiked(comment, user)
	if err != nil {
		return nil, err
	}

	return like, nil
}

func (m *CommentModel) UserHasLiked(comment *Comment, user *User) (bool, error) {
	if user == nil || comment == nil {
		return false, nil
	}

	query := `
		SELECT 1
		FROM comment_like
		WHERE user_id = $1 AND comment_id = $2
		LIMIT 1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var exists int
	err := m.DB.QueryRowContext(ctx, query, user.ID, comment.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func (m *CommentModel) Counts(articles []*Article) (map[int]int, error) {
	counts := make(map[int]int)

	for _, article := range articles {
		if _, ok := counts[article.ID]; ok {
			continue
		}

		count, err := m.Count(article)

		if err != nil {
			return nil, err
		}
		counts[article.ID] = count
	}

	return counts, nil
}
