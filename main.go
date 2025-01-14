package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/starshine-sys/pkgo/v2"
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

type config struct {
	SystemID string
}

func main() {
	// read config file
	content, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	var loadedConfig config
	err = json.Unmarshal(content, &loadedConfig)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	// read dotenv file
	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}

	pkAuth := os.Getenv("pluralkit_auth")

	// pk api object thingy
	pk := pkgo.New(pkAuth)
	//load our system
	sys, err := pk.System(loadedConfig.SystemID)
	if err != nil {
		log.Fatal("Error loading system")
	}
	log.Default().Print(sys)

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

	app.Get("/system", func(c *fiber.Ctx) error {
		return c.SendString(sys.Name)
	})

	log.Fatal(app.Listen(":5080"))
}
