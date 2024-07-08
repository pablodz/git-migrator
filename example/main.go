package main

import (
	"log"

	"github.com/pablodz/git-migrator/migrator"
)

func main() {

	err := migrator.MigrateToFakeCommitRepo("/abolsute/path/origin", "/absolute/path/destiny")
	if err != nil {
		log.Fatal(err)
	}
}
