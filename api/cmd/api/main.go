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

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rberrelleza/cloud-native-recipes/api/pkg/database"
	"github.com/rberrelleza/cloud-native-recipes/api/pkg/stats"
)

type Server struct {
	ctx context.Context
	db  *database.Database
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uri := fmt.Sprintf("mongodb://%s:27017", os.Getenv("MONGODB_HOST"))

	db, err := database.Connect(ctx, uri, os.Getenv("MONGODB_DATABASE"), os.Getenv("MONGODB_USERNAME"), os.Getenv("MONGODB_PASSWORD"))
	if err != nil {
		log.Fatal(err)
	}

	defer db.Disconnect()
	log.Println("connected to mongodb")

	srv := Server{
		ctx: ctx,
		db:  db,
	}

	router := mux.NewRouter()
	router.Use(jsonContentTypeMiddleware)
	router.Use(stats.WrapHTTPHandler)
	router.HandleFunc("/api/healthz", srv.healthcheckHandler)
	router.HandleFunc("/api/recipes", srv.recipesHandler)
	router.HandleFunc("/api/recipes/{id}", srv.recipeHandler)
	router.HandleFunc("/api/recipes/{id}/up", srv.upvoteHandler)
	router.HandleFunc("/api/recipes/{id}/down", srv.downvoteHandler)
	router.Handle("/metrics", promhttp.Handler())

	// catch shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, os.Interrupt, os.Kill)

	// Start the api server.
	go func() {
		log.Fatal(http.ListenAndServe(":8080", router))
	}()

	sig := <-sigs
	log.Println("exiting with signal: ", sig)
}

func (s *Server) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) recipesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipes, err := s.db.GetRecipes(ctx)
	if err != nil {
		log.Printf("error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, recipes)
}

func (s *Server) recipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]
	if recipeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	recipe, err := s.db.GetRecipe(ctx, recipeID)
	if err != nil {
		log.Printf("error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	respondWithJSON(w, http.StatusOK, recipe)
}

func (s *Server) upvoteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]
	if recipeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	recipe, err := s.db.UpVoteRecipe(ctx, recipeID)
	if err != nil {
		log.Printf("error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	respondWithJSON(w, http.StatusOK, recipe)
}

func (s *Server) downvoteHandler(w http.ResponseWriter, r *http.Request) {
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

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json; charset=UTF-8")
		next.ServeHTTP(w, r)
	})
}
