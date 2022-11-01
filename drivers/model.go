package drivers

import (
	"errors"
	"log"
	"net/url"
	"os"

	"github.com/pbnjay/bdog"
	"github.com/pbnjay/bdog/drivers/sqlite3"
)

func Init(dbName string) (bdog.Model, error) {
	uu, err := url.Parse(dbName)
	if err == nil {
		log.Println("DB Scheme: ", uu.Scheme)
	}

	info, err := os.Stat(dbName)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("drivers: cannot introspect a directory")
	}

	return sqlite3.Open(dbName)
}
