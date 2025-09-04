package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	// Sets upload limit to 1GB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	// Extract videoID from URL path params and parse as a UUID
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse videoID", err)
		return
	}

	// Authenticate user to get a userID
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

	// Get video metadata from the database
	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error retrieving video metadata", err)
		return
	}
	if userID != dbVideo.UserID {
		respondWithError(w, http.StatusUnauthorized, "Access Denied", nil)
	}

	// Parse uploaded video file from the form data
	fileData, _, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error retrieving video", err)
		return
	}
	defer fileData.Close()

	// Validate uploaded file to ensure it's an MP4
	_, _, err = mime.ParseMediaType("video/mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}

	// Save uploaded file to a temp file on disk
	file, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error creating temp file", err)
		return
	}
	defer os.Remove(file.Name())
	defer file.Close()

	_, err = io.Copy(file, fileData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error copying data to file", err)
		return
	}

	// Reset tempFile's file pointer to the beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error seeking start", err)
		return
	}

	// Generate key
	byteSlice := make([]byte, 32)
	num, err := rand.Read(byteSlice)
	if num != 32 {
		respondWithError(w, http.StatusBadRequest, "Byte slice was not filled fully", nil)
		return
	} else if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error filling byte slice", err)
		return
	}

	fileName := base64.RawURLEncoding.EncodeToString(byteSlice)
	key := fmt.Sprintf("%v.mp4", fileName)

	// Put object into S3
	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String("tubely-6543210"),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String("video/mp4"),
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error putting object in s3", err)
		return
	}

	// Update videoURL in database with s3 bucket and key
	s3URL := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, key)

	dbVideo.VideoURL = &s3URL
	cfg.db.UpdateVideo(dbVideo)
}
