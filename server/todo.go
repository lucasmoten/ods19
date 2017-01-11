package server

import (
	"net/http"

	"golang.org/x/net/context"
)

// Unimplemented routes

func (h AppServer) getRelationships(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "getRelationships is not yet implemented")
}

func (h AppServer) removeObjectSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "removeObjectSubscription is not yet implemented")
}

func (h AppServer) removeObjectFromFolder(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "removeObjectFromFolder is not yet implemented")
}

func (h AppServer) removeObjectFromFavorites(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "removeObjectFromFavorites is not yet implemented")
}

func (h AppServer) addObjectToFolder(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "addObjectToFolder is not yet implemented")
}

func (h AppServer) listFavorites(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "listFavorites is not yet implemented")
}
