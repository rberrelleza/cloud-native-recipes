package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rberrelleza/cloud-native-recipes/api/pkg/database"
)

//go:embed recipes.json
var recipesJSON []byte

func main() {
	log.Println("loading data")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uri := fmt.Sprintf("mongodb://%s:27017", os.Getenv("MONGODB_HOST"))

	db, err := database.Connect(ctx, uri, os.Getenv("MONGODB_DATABASE"), os.Getenv("MONGODB_USERNAME"), os.Getenv("MONGODB_PASSWORD"))
	if err != nil {
		log.Fatal(err)
	}

	defer db.Disconnect()
	log.Println("connected to mongodb")

	var recipes []database.Recipe

	if err := json.Unmarshal(recipesJSON, &recipes); err != nil {
		log.Fatal(err)
	}

	if err := db.Load(ctx, recipes); err != nil {
		log.Fatalf("error: %s", err)
	}

	log.Println("loaded data")
}
