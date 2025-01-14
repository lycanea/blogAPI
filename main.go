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
		// Retrieve system information
		sys, err := pk.System(loadedConfig.SystemID)
		if err != nil {
			log.Println("Error loading system:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to load system information",
			})
		}

		// Construct the systemInfo JSON object
		systemInfo := fiber.Map{
			"id":          sys.ID,
			"name":        sys.Name,
			"avatar":      sys.AvatarURL,
			"banner":      sys.Banner,
			"color":       sys.Color,
			"created":     sys.Created,
			"description": sys.Description,
			"tag":         sys.Tag,
		}

		// Return the JSON response
		return c.JSON(systemInfo)
	})

	app.Get("/system/list", func(c *fiber.Ctx) error {
		members, err := pk.Members(loadedConfig.SystemID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch members",
			})
		}

		var membersList []map[string]interface{}

		for _, member := range members {
			// Assuming each member has fields Name, Description, etc.
			membersList = append(membersList, map[string]interface{}{
				"display": member.DisplayName,
				"name":    member.Name,
				"id":      member.ID,
				// Add other fields you want to include
			})
		}

		return c.JSON(membersList)
	})

	app.Get("/system/member/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		member, err := pk.Member(idParam)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch member",
			})
		}

		memberInfo := fiber.Map{
			"display":     member.DisplayName,
			"name":        member.Name,
			"description": member.Description,
			"id":          member.ID,
			"created":     member.Created,
			"color":       member.Color,
			"avatar":      member.AvatarURL,
			"banner":      member.Banner,
			"birthday":    member.Birthday,
			"pronouns":    member.Pronouns,
		}

		return c.JSON(memberInfo)
	})

	log.Fatal(app.Listen(":8080"))
}
