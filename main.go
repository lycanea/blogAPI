package main

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Post struct {
	ID      string   `json:"id" bson:"_id"`
	Post_ID int32    `json:"post_id" bson:"_post_id_"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Date    int32    `json:"date"`
	Author  string   `json:"author"`
	Color   []string `json:"colors"`
}

func main() {
	// MongoDB connection
	client, err := mongo.Connect(nil, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(nil)

	err = client.Ping(nil, readpref.Primary())
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	db := client.Database("blog")
	postsCollection := db.Collection("posts")

	// Fiber app setup
	app := fiber.New()

	app.Get("/posts", func(c *fiber.Ctx) error {
		cursor, err := postsCollection.Find(nil, bson.D{})
		if err != nil {
			return c.Status(500).SendString("Database query failed")
		}
		defer cursor.Close(nil)

		var posts []Post
		if err := cursor.All(nil, &posts); err != nil {
			log.Default().Print(posts)
			return c.Status(500).SendString("Failed to decode posts")
		}
		return c.JSON(posts)
	})

	app.Get("/posts/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		// Convert idParam to int32
		id, err := strconv.ParseInt(idParam, 10, 32)
		if err != nil {
			log.Default().Print("Invalid ID format:", idParam)
			return c.Status(400).SendString("Invalid post ID")
		}

		// Use int32 value for the query
		var post Post
		err = postsCollection.FindOne(nil, bson.M{"_post_id_": int32(id)}).Decode(&post)
		if err != nil {
			log.Default().Print("Post not found:", id)
			return c.Status(404).SendString("Post not found")
		}

		return c.JSON(post)
	})

	log.Fatal(app.Listen(":8080"))
}
