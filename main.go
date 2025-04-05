package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"github.com/patrickmn/go-cache"
	"github.com/starshine-sys/pkgo/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type memberCache struct {
	members *cache.Cache
}

const (
	defaultExpiration = 5 * time.Minute
	purgeTime         = 10 * time.Minute
)

func newCache() *memberCache {
	Cache := cache.New(defaultExpiration, purgeTime)
	return &memberCache{
		members: Cache,
	}
}

func (pkcache *memberCache) read(id string) (item pkgo.Member, ok bool) {
	member, ok := pkcache.members.Get(id)
	if ok {
		log.Println("from cache")
		res := member.(pkgo.Member)
		if !ok {
			return pkgo.Member{}, false
		}
		return res, true
	}
	return pkgo.Member{}, false
}

func (pkcache *memberCache) update(id string, member pkgo.Member) {
	log.Printf("Updating cache for member ID: %s", id)
	pkcache.members.Set(id, member, cache.DefaultExpiration)
}

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

type PostResponse struct {
	ID      string   `json:"id"`
	Post_ID int32    `json:"post_id" bson:"_post_id_"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
	Author  string   `json:"author"`
	Color   []string `json:"colors"`
}

type config struct {
	SystemID string
}

func main() {
	var pkcache = newCache()

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
	client, err := mongo.Connect(nil,
		options.Client().ApplyURI("mongodb://localhost:27017"))
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

	// Configure CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // i think this is like websites that this is allowed to be requested from
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "*",
	}))

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

		var response []PostResponse
		for _, post := range posts {
			response = append(response, PostResponse{
				ID:      post.ID,
				Post_ID: post.Post_ID,
				Title:   post.Title,
				Tags:    post.Tags,
				Author:  post.Author,
				Color:   post.Color,
			})
		}

		return c.JSON(response)
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
		err = postsCollection.FindOne(nil, bson.M{"_post_id_": int32(id)}).
			Decode(&post)
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
			if member.Privacy.Visibility != "private" {
				membersList = append(membersList, map[string]interface{}{
					"display": member.DisplayName,
					"name":    member.Name,
					"id":      member.ID,
					// Add other fields you want to include
				})
			}
		}

		return c.JSON(membersList)
	})

	app.Get("/system/member/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		// fmt.Println(pkcache.members.Items())
		// cached, ok := pkcache.read(idParam)
		// if ok {
		// 	if cached.Privacy.Visibility == "private" {
		// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		// 			"error": "Failed to fetch member",
		// 		})
		// 	}
		// 	memberInfo := fiber.Map{
		// 		"display":     cached.DisplayName,
		// 		"name":        cached.Name,
		// 		"description": cached.Description,
		// 		"id":          cached.ID,
		// 		"created":     cached.Created,
		// 		"color":       cached.Color,
		// 		"avatar":      cached.AvatarURL,
		// 		"banner":      cached.Banner,
		// 		"birthday":    cached.Birthday,
		// 		"pronouns":    cached.Pronouns,
		// 	}
		// 	return c.JSON(memberInfo)
		// }
		member, err := pk.Member(idParam)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch member",
			})
		}
		if member.Privacy.Visibility == "private" {
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
		pkcache.update(idParam, member)
		return c.JSON(memberInfo)
	})

	app.Get("/modded/message", func(c *fiber.Ctx) error {
		return c.SendString("cuboid now owns/runs the server, i (lycanea) might help with some stuff tho idk meow")
	})

	log.Fatal(app.Listen(":8080"))
}
