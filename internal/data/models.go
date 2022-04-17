package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound  = errors.New("record not found")
	ErrDuplicateRecord = errors.New("duplicate record")
	ErrEditConflict    = errors.New("edit conflict")
)

type Models struct {
	Users        UserModel
	Publications PublicationModel
	Articles     ArticleModel
	Images       ImageModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:        UserModel{DB: db},
		Publications: PublicationModel{DB: db},
		Articles:     ArticleModel{DB: db},
		Images:       ImageModel{DB: db},
	}
}
