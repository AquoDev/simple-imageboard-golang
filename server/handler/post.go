package handler

import (
	"encoding/json"
	"time"

	"github.com/AquoDev/simple-imageboard-golang/database"
	"github.com/AquoDev/simple-imageboard-golang/database/methods"
	"github.com/AquoDev/simple-imageboard-golang/redis"
	"github.com/AquoDev/simple-imageboard-golang/server/utils"
	"github.com/asaskevich/govalidator"
	"github.com/kataras/iris"
)

// GetPostExample handles a JSON response with a how-to example.
func GetPostExample(ctx iris.Context) {
	ctx.JSON(map[string]string{
		"info":    "If you want to see any post, see the example.",
		"example": "{url}/post/{id}",
	})
}

// GetPost handles a JSON response with a post.
func GetPost(ctx iris.Context) {
	var response interface{}
	var err error

	// Parse post ID
	id := ctx.Params().GetUint64Default("id", 0)
	redisKey := redis.GetPostKey(id)

	// Get post from cache
	response, err = redis.Client().Get(redisKey).Result()
	// If it exists, return a response with it
	if err == nil {
		post := new(database.Post)
		cachedPost := []byte(response.(string))
		json.Unmarshal(cachedPost, &post)
		ctx.JSON(post)
		return
	}

	// If it couldn't be found in cache, get it from the database
	response, err = methods.GetPost(id)
	if err != nil {
		// If the post isn't found, the response will be 404 Not Found
		response = GetError(404)
	}

	// Set the cache
	cachedPost, _ := json.Marshal(response)
	redis.Client().Set(redisKey, string(cachedPost), 30*time.Second)

	ctx.JSON(response)
}

// SavePost handles a JSON response and saves the data as a post.
func SavePost(ctx iris.Context) {
	post := new(database.Post)

	// Read JSON from body
	if err := ctx.ReadJSON(&post); err != nil {
		invalidData := GetError(400)
		ctx.JSON(invalidData)
		return
	}

	// Check if content has characters
	if len(post.Content) == 0 {
		invalidData := GetError(400)
		ctx.JSON(invalidData)
		return
	}

	// Check if it has already an ID (it shouldn't have)
	if post.ID != 0 {
		invalidData := GetError(400)
		ctx.JSON(invalidData)
		return
	}

	// Check if pic exists and it's an invalid URL (it should be a valid URL)
	if post.Pic != nil && !govalidator.IsURL(post.Pic.String) {
		invalidData := GetError(400)
		ctx.JSON(invalidData)
		return
	}

	// Make delete code if it hasn't one
	if post.DeleteCode == "" {
		post.DeleteCode = utils.RandomString(32)
	}

	// Try to save the post (or thread) and check if it has been saved
	if response, err := methods.SavePost(post); err != nil {
		invalidData := GetError(400)
		ctx.JSON(invalidData)
	} else {
		if post.OnThread == nil {
			// Delete old threads when the post starts a new thread
			methods.DeleteOldThreads()
		} else {
			// Bump thread if it hasn't reached bump limit
			methods.BumpThread(uint64(post.OnThread.ValueOrZero()), post.CreatedAt)
		}
		ctx.JSON(response)
	}
}

// DeletePost handles a JSON response with a post.
func DeletePost(ctx iris.Context) {
	data := new(utils.DeleteData)

	// Read JSON from body
	if err := ctx.ReadJSON(&data); err != nil {
		invalidData := GetError(400)
		ctx.JSON(invalidData)
		return
	}

	// Try to get post from database to check if it exists
	if _, err := methods.GetPost(data.ID); err != nil {
		postDoesntExist := GetError(404)
		ctx.JSON(postDoesntExist)
		return
	}

	// Try to delete post (and thread if the post has "on_thread == null")
	if err := methods.DeletePost(data.ID, data.DeleteCode); err != nil {
		incorrectCode := GetError(400)
		ctx.JSON(incorrectCode)
		return
	}

	ctx.JSON(map[string]interface{}{
		"status":  200,
		"message": "Success",
	})
}
