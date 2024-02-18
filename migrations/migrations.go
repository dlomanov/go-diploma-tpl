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
	{Name: "m0001.sql", Title: "M0001: Users table", NoTx: false},
	{Name: "m0002.sql", Title: "M0002: Balances table", NoTx: false},
	{Name: "m0003.sql", Title: "M0003: Orders table", NoTx: false},
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
