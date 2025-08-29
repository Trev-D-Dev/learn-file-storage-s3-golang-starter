package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	// Max Memory of 10MB, bit shifted left 20 times to get number of bytes
	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	// Get the image data from the form
	fileData, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error retrieving thumbnail", err)
		return
	}
	mediaType := fileHeader.Header.Get("Content-Type")

	// Read image data into a byte slice
	imageData, err := io.ReadAll(fileData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error reading from file data", err)
		return
	}

	// Get video's metadata from the SQLite database
	dbVideo, err := cfg.db.GetVideo(videoID)
	if userID != dbVideo.UserID {
		respondWithError(w, http.StatusUnauthorized, "Access Denied", err)
		return
	}

	// Save the thumbnail to the global map
	thumb := thumbnail{
		data:      imageData,
		mediaType: mediaType,
	}
	videoThumbnails[videoID] = thumb

	// Update the video metadata so it has a new thumbnail URL
	thumbURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, videoID)
	dbVideo.ThumbnailURL = &thumbURL
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error updating video in database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
