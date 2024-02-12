package migrations

import (
	"embed"
)

type Migration struct {
	Name  string
	Query string
	NoTx  bool
}

type file struct {
	Name  string
	Title string
	NoTx  bool
}

//go:embed *
var fs embed.FS
var files = []file{
	{Name: "m0001.sql", Title: "M0001: First migrator", NoTx: false},
}

func GetMigrations() ([]Migration, error) {
	result := make([]Migration, len(files))

	for i, f := range files {
		query, err := fs.ReadFile(f.Name)
		if err != nil {
			return nil, err
		}

		result[i] = Migration{
			Name:  f.Title,
			Query: string(query),
			NoTx:  f.NoTx,
		}
	}

	return result, nil
}
