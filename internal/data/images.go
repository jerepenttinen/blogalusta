package data

import (
	"context"
	"database/sql"
	"time"
)

type ImageModel struct {
	DB *sql.DB
}

func (m *ImageModel) Get(id int) ([]byte, error) {
	query := `
		SELECT image_data
		FROM image
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var image []byte
	err := m.DB.QueryRowContext(ctx, query, id).Scan(&image)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return image, nil
}

func (m *ImageModel) Insert(image []byte) (int, error) {
	query := `
		INSERT INTO image (image_data)
		VALUES ($1)
		RETURNING id`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	id := 0
	err := m.DB.QueryRowContext(ctx, query, image).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}
