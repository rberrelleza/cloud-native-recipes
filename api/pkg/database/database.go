package database

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/avast/retry-go"
	"github.com/google/uuid"
	multierror "github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	ctx        context.Context
	client     *mongo.Client
	collection *mongo.Collection
}

type Recipe struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Image     string `json:"image"`
	UpVotes   int    `json:"upVotes"`
	DownVotes int    `json:"downVotes"`
}

func Connect(ctx context.Context, uri, database, username, password string) (*Database, error) {
	credential := options.Credential{
		AuthSource: database,
		Username:   username,
		Password:   password,
	}

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

			return nil
		},
		retry.Attempts(3),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("failed to connect to mongo #%d: %s\n", n, err)
		}),
	)

	if err != nil {
		return nil, err
	}

	return &Database{client: client, collection: client.Database(database).Collection("recipes")}, nil
}

func (d *Database) Disconnect() {
	if d.client != nil {
		d.client.Disconnect(d.ctx)
	}
}

func (d *Database) GetRecipes(ctx context.Context) ([]Recipe, error) {
	cur, err := d.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("can't get recipes: %w", err)
	}

	defer cur.Close(ctx)
	var recipes []Recipe
	if err := cur.All(ctx, &recipes); err != nil {
		return nil, fmt.Errorf("can't decode recipes: %w", err)
	}

	sort.Slice(recipes, func(i, j int) bool {

		return strings.Compare(recipes[i].Title, recipes[j].Title) == -1
	})

	return recipes, nil
}

func (d *Database) GetRecipe(ctx context.Context, recipeID string) (*Recipe, error) {
	cur := d.collection.FindOne(ctx, bson.M{"id": recipeID})
	if cur.Err() != nil {
		return nil, fmt.Errorf("can't get recipe %s: %w", recipeID, cur.Err())
	}

	var result Recipe
	if err := cur.Decode(&result); err != nil {
		return nil, fmt.Errorf("can't decode recipe %s: %w", recipeID, err)
	}

	return &result, nil
}

func (d *Database) UpVoteRecipe(ctx context.Context, recipeID string) (*Recipe, error) {
	filter := bson.M{"id": recipeID}
	update := bson.M{"$inc": bson.M{"upVotes": 1}}
	result := d.collection.FindOneAndUpdate(ctx, filter, update)
	if result.Err() != nil {
		return nil, fmt.Errorf("can't get recipe %s: %w", recipeID, result.Err())
	}

	var recipe Recipe
	if err := result.Decode(&recipe); err != nil {
		return nil, fmt.Errorf("can't decode recipe %s: %w", recipeID, err)
	}

	// hack to prevent querying again
	recipe.UpVotes++
	return &recipe, nil
}

func (d *Database) DownVoteRecipe(ctx context.Context, recipeID string) (*Recipe, error) {
	filter := bson.M{"id": recipeID}
	update := bson.M{"$inc": bson.M{"downVotes": 1}}
	result := d.collection.FindOneAndUpdate(ctx, filter, update)
	if result.Err() != nil {
		return nil, fmt.Errorf("can't get recipe %s: %w", recipeID, result.Err())
	}

	var recipe Recipe
	if err := result.Decode(&recipe); err != nil {
		return nil, fmt.Errorf("can't decode recipe %s: %w", recipeID, err)
	}

	// hack to prevent querying again
	recipe.DownVotes++
	return &recipe, nil
}

func (d *Database) AddRecipe(ctx context.Context, recipe Recipe) (*Recipe, error) {

	if recipe.ID == "" {
		recipe.ID = uuid.New().String()
	}

	_, err := d.collection.InsertOne(ctx, recipe)
	if err != nil {
		return nil, fmt.Errorf("can't insert recipe: %s", err)
	}

	return d.GetRecipe(ctx, recipe.ID)

}

func (d *Database) Load(ctx context.Context, recipes []Recipe) error {
	var result error
	opts := options.Update().SetUpsert(true)
	for _, r := range recipes {
		filter := bson.M{"id": r.ID}
		update := bson.M{"$set": r}
		_, err := d.collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}
