module blogalusta

// +heroku goVersion go1.18
// +heroku install -tags 'postgres' ./vendor/github.com/golang-migrate/migrate/v4/cmd/migrate .
go 1.18

require (
	github.com/go-chi/chi/v5 v5.0.7
	github.com/lib/pq v1.10.4
)
