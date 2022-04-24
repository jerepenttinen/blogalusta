package data

import (
	"context"
	"database/sql"
	"errors"
	"github.com/gosimple/slug"
	"golang.org/x/crypto/bcrypt"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	CreatedAt      time.Time
	Version        int
	ImageID        sql.NullInt64
}

func (u *User) Matches(url string) bool {
	return url == slug.Make(u.Name)
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(name, email, password string) (int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return 0, err
	}

	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	id := 0
	err = m.DB.QueryRowContext(ctx, query, name, email, hashedPassword).Scan(&id)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return 0, ErrDuplicateRecord
		default:
			return 0, err
		}
	}

	return id, nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	var id int
	var hashedPassword []byte

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := m.DB.QueryRowContext(ctx, `SELECT id, password_hash FROM users WHERE email = $1`, email)
	err := row.Scan(&id, &hashedPassword)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidCredentials
	} else if err != nil {
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, ErrInvalidCredentials
	} else if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *UserModel) Get(id int) (*User, error) {
	s := &User{}

	stmt := `SELECT id, name, email, created_at, image_id, version FROM users WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, id).Scan(&s.ID, &s.Name, &s.Email, &s.CreatedAt, &s.ImageID, &s.Version)

	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return s, nil
}

func (m *UserModel) GetWritersOfPublication(publication *Publication) ([]*User, error) {

	stmt := `
		SELECT id, name, email, created_at, image_id
		FROM writes_on
		JOIN users on id = writes_on.user_id
		WHERE publication_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var users []*User
	rows, err := m.DB.QueryContext(ctx, stmt, publication.ID)
	for rows.Next() {
		u := &User{}
		err = rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.ImageID)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return users, nil
}

func (m *UserModel) ChangeProfilePicture(user *User, id int) error {
	query := `
		UPDATE users
		SET image_id = $1, version = version + 1
		WHERE id = $2 AND version = $3`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id, user.ID, user.Version)
	if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) SubscribeTo(user *User, publication *Publication) error {
	query := `
		INSERT INTO subscribes_to (user_id, publication_id)
		VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user.ID, publication.ID)
	if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) UnsubscribeFrom(user *User, publication *Publication) error {
	query := `
		DELETE FROM subscribes_to
		WHERE user_id = $1 AND publication_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user.ID, publication.ID)
	if err == sql.ErrNoRows {
		return ErrRecordNotFound
	} else if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) GetByEmail(email string) (*User, error) {
	s := &User{}

	stmt := `SELECT id, name, email, created_at, image_id FROM users WHERE email = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, email).Scan(&s.ID, &s.Name, &s.Email, &s.CreatedAt, &s.ImageID)

	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return s, nil
}

func (m *UserModel) Invitations(user *User) ([]*Publication, error) {

	stmt := `
		SELECT id, name, url, description, owner_id, created_at
		FROM invitation
		JOIN publication on id = invitation.publication_id
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var publications []*Publication
	rows, err := m.DB.QueryContext(ctx, stmt, user.ID)
	for rows.Next() {
		p := &Publication{}
		err = rows.Scan(&p.ID, &p.Name, &p.URL, &p.Description, &p.OwnerID, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		publications = append(publications, p)
	}

	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return publications, nil
}

func (m *UserModel) AcceptInvitation(user *User, publicationID int) error {
	stmt := `
		DELETE 
		FROM invitation
		WHERE user_id = $1 AND publication_id = $2`

	ctx0, cancel0 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel0()

	_, err := m.DB.ExecContext(ctx0, stmt, user.ID, publicationID)

	if err == sql.ErrNoRows {
		return ErrRecordNotFound
	} else if err != nil {
		return err
	}

	stmt = `
		INSERT INTO writes_on (user_id, publication_id)
		VALUES ($1, $2)`

	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()

	_, err = m.DB.ExecContext(ctx1, stmt, user.ID, publicationID)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "writes_on_pk"` {
			return ErrDuplicateRecord
		}
		return err
	}

	return nil
}

func (m *UserModel) DeclineInvitation(user *User, publicationID int) error {
	stmt := `
		DELETE 
		FROM invitation
		WHERE user_id = $1 AND publication_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, stmt, user.ID, publicationID)

	if err == sql.ErrNoRows {
		return ErrRecordNotFound
	} else if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) Leave(user *User, publicationID int) error {
	stmt := `
		DELETE 
		FROM writes_on
		WHERE user_id = $1 AND publication_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, stmt, user.ID, publicationID)

	if err == sql.ErrNoRows {
		return ErrRecordNotFound
	} else if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) ArticleWriters(articles []*Article) (map[int]*User, error) {
	writers := make(map[int]*User)

	for i := range articles {
		id := articles[i].WriterID
		if _, ok := writers[id]; ok {
			continue
		}

		writer, err := m.Get(id)
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		} else if err != nil {
			return nil, err
		}

		writers[id] = writer
	}
	return writers, nil
}

func (m *UserModel) PublicationOwners(p []*Publication) (map[int]*User, error) {
	owners := make(map[int]*User)

	for i := range p {
		id := p[i].OwnerID
		if _, ok := owners[id]; ok {
			continue
		}

		owner, err := m.Get(id)
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		} else if err != nil {
			return nil, err
		}

		owners[id] = owner
	}
	return owners, nil
}

func (m *UserModel) LikeArticle(user *User, article *Article) error {
	query := `
		INSERT INTO article_like (user_id, article_id)
		VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user.ID, article.ID)
	if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) UnlikeArticle(user *User, article *Article) error {
	query := `
		DELETE FROM article_like
		WHERE user_id = $1 AND article_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user.ID, article.ID)
	if err == sql.ErrNoRows {
		return ErrRecordNotFound
	} else if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) LikeComment(user *User, comment *Comment) error {
	query := `
		INSERT INTO comment_like (user_id, comment_id)
		VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user.ID, comment.ID)
	if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) UnlikeComment(user *User, comment *Comment) error {
	query := `
		DELETE FROM comment_like
		WHERE user_id = $1 AND comment_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user.ID, comment.ID)
	if err == sql.ErrNoRows {
		return ErrRecordNotFound
	} else if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) ChangeName(user *User, name string) error {
	query := `
		UPDATE users
		SET name = $1, version = version + 1
		WHERE users.id = $2 AND version = $3`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, name, user.ID, user.Version)
	if err == sql.ErrNoRows {
		return ErrEditConflict
	} else if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) ChangePassword(user *User, oldPass, newPass string) error {
	var hashedPassword []byte

	query := `
		SELECT password_hash
		FROM users
		WHERE id = $1 AND version = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, user.ID, user.Version).Scan(&hashedPassword)
	if err == sql.ErrNoRows {
		return ErrEditConflict
	} else if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(oldPass))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return ErrInvalidCredentials
	} else if err != nil {
		return err
	}

	hashedPassword, err = bcrypt.GenerateFromPassword([]byte(newPass), 12)
	if err != nil {
		return err
	}

	query = `
		UPDATE users
		SET password_hash = $1, version = version + 1
		WHERE id = $2 AND version = $3`

	_, err = m.DB.ExecContext(ctx, query, hashedPassword, user.ID, user.Version)
	if err == sql.ErrNoRows {
		return ErrEditConflict
	} else if err != nil {
		return err
	}

	return nil
}

func (m *UserModel) HasPublication(user *User) (bool, error) {
	query := `
		SELECT 1
		FROM writes_on
		WHERE user_id = $1
		LIMIT 1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	exists := 0
	err := m.DB.QueryRowContext(ctx, query, user.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func (m *UserModel) HasInvitations(user *User) (bool, error) {
	query := `
		SELECT 1
		FROM invitation
		WHERE user_id = $1
		LIMIT 1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	exists := 0
	err := m.DB.QueryRowContext(ctx, query, user.ID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return exists == 1, nil
}
