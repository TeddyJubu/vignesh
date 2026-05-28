//go:build ignore

package main

import (
	"fmt"
	"os"

	"ai-receptionist/internal/store"
)

func main() {
	dbPath := "/opt/ai-receptionist/database.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	provider := "ollama"
	if len(os.Args) > 2 {
		provider = os.Args[2]
	}
	db, err := store.Open(dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.UpsertAppSetting("ai.provider", provider); err != nil {
		panic(err)
	}
	v, _ := db.GetAppSetting("ai.provider")
	fmt.Println("ai.provider=", v)
}
