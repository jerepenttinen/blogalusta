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
	ID             int64
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

func (m *UserModel) Insert(name, email, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = m.DB.ExecContext(ctx, query, name, email, hashedPassword)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}

	return nil
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

	stmt := `SELECT id, name, email, created_at, image_id FROM users WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, id).Scan(&s.ID, &s.Name, &s.Email, &s.CreatedAt, &s.ImageID)

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
		SET image_id = $1
		WHERE id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id, user.ID)
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

func (m *UserModel) GetPendingInvitations(publication *Publication) ([]*User, error) {

	stmt := `
		SELECT id, name, email, created_at, image_id
		FROM invitation
		JOIN users on id = invitation.user_id
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
