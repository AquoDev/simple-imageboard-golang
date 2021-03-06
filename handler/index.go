package handler

import (
	"net/http"

	"github.com/AquoDev/simple-imageboard-golang/cache"
	"github.com/AquoDev/simple-imageboard-golang/database"
	"github.com/AquoDev/simple-imageboard-golang/framework"
)

// GetIndex handles a JSON response with every post that started a thread.
func GetIndex(ctx framework.Context) error {
	// Try to get the index from cache. Is it stored in cache?
	// ── Yes:	check if what's cached is an error or a list. Both options lead to a response.
	// ── No:	continue.
	if response, err := cache.GetCachedIndex(); err == nil {
		// Is the list cached?
		// ── Yes:	return the cached list.
		// ── No:	it's an error, so return that error as a failed JSON response.
		if response.Status == http.StatusOK {
			return framework.SendOK(ctx, response.Data)
		}
		return framework.SendError(response.Status)
	}

	// Try to get the index from the database, even if the slice we get is empty. Did it go correctly?
	// ── Yes:	cache the list and return it, even if it's empty.
	// ── No:	continue. There must be a server-side error. This means something has gone seriously wrong.
	if response, err := database.GetIndex(); err == nil {
		go cache.SetCachedIndex(http.StatusOK, response)
		return framework.SendOK(ctx, response)
	}

	// Return a 500 InternalServerError JSON response after caching it.
	go cache.SetCachedIndex(http.StatusInternalServerError, nil)
	return framework.SendError(http.StatusInternalServerError)
}
