package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var collection *mongo.Collection

type Recipe struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Image     string `json:"image"`
	UpVotes   int    `json:"upVotes"`
	DownVotes int    `json:"downVotes"`
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json; charset=UTF-8")
		next.ServeHTTP(w, r)
	})
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	credential := options.Credential{
		AuthSource: os.Getenv("MONGODB_DATABASE"),
		Username:   os.Getenv("MONGODB_USERNAME"),
		Password:   os.Getenv("MONGODB_PASSWORD"),
	}

	uri := fmt.Sprintf("mongodb://%s:27017", os.Getenv("MONGODB_HOST"))

	clientOpts := options.Client().ApplyURI(uri).SetAuth(credential)

	var client *mongo.Client
	err := retry.Do(
		func() error {
			c, err := mongo.Connect(ctx, clientOpts)
			if err != nil {
				return err
			}

			if err := c.Ping(ctx, readpref.Primary()); err != nil {
				return err
			}

			client = c
			collection = c.Database(os.Getenv("MONGODB_DATABASE")).Collection("recipes")
			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			log.Printf("failed to connect to mongo #%d: %s\n", n, err)
		}),
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("connected to mongodb")

	defer client.Disconnect(ctx)

	router := mux.NewRouter()
	router.Use(jsonContentTypeMiddleware)
	router.HandleFunc("/api/healthz", healthcheckHandler)
	router.HandleFunc("/api/recipes", recipesHandler)
	router.HandleFunc("/api/recipes/{id}", recipeHandler)
	router.HandleFunc("/api/recipes/{id}/up", upvoteHandler)
	router.HandleFunc("/api/recipes/{id}/down", downvoteHandler)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, os.Interrupt, os.Kill)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", router))
	}()

	sig := <-sigs
	log.Println("exiting with signal: ", sig)

}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	if collection == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func recipesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Printf("error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer cur.Close(ctx)
	var recipes []Recipe
	if err := cur.All(ctx, &recipes); err != nil {
		log.Printf("decoding error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, recipes)
}

func recipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]
	if recipeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	cur := collection.FindOne(ctx, bson.M{"id": recipeID})
	if cur.Err() != nil {
		log.Printf("recipe not found: %s\n", cur.Err().Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result Recipe
	if err := cur.Decode(&result); err != nil {
		log.Printf("decoding error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func upvoteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]
	if recipeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	filter := bson.M{"id": recipeID}
	update := bson.M{"$inc": bson.M{"upVotes": 1}}
	cur := collection.FindOneAndUpdate(ctx, filter, update)
	if cur.Err() != nil {
		log.Printf("recipe not found: %s\n", cur.Err().Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result Recipe
	if err := cur.Decode(&result); err != nil {
		log.Printf("decoding error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// hack to prevent querying again
	result.UpVotes++
	respondWithJSON(w, http.StatusOK, result)
}

func downvoteHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("marshalling error: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)
	w.Write(response)
}
