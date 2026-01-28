package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video", err)
		return 
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized attempt", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return 
	}

	defer file.Close()

	media_type, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to parse media type", err)
		return 
	}

	if media_type != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Not the required media type", nil)
		return 
	}

	temp_file, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create temp file", err)
		return 
	}

	defer os.Remove(temp_file.Name())
	defer temp_file.Close()

	_, err = io.Copy(temp_file, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to copy file", err)
		return
	}

	processed_video, err := processVideoForFastStart(temp_file.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	processed_file, err := os.Open(processed_video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to open file", err)
		return
	}

	defer os.Remove(processed_file.Name())
	defer processed_file.Close()

	info, err := processed_file.Stat()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to Retrieve metadata", err)
		return
	}
	size := info.Size()

	aspectRatio, err := getVideoAspectRatio(temp_file.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error determining aspect ratio", err)
		return
	}

	directory := ""
	switch aspectRatio {
	case "16:9":
		directory = "landscape"
	case "9:16":
		directory = "portrait"
	default:
		directory = "other"
	}

	_, err = temp_file.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reset temp file", err)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)
	rand_file_name := base64.RawURLEncoding.EncodeToString(key)
	file_type := strings.Split(media_type, "/")
	image_path := rand_file_name + "." + file_type[1]
	image_path = path.Join(directory, image_path)

	_, err = cfg.s3Client.PutObject(
		r.Context(),
		&s3.PutObjectInput{
			Bucket: aws.String(cfg.s3Bucket),
			Key: aws.String(image_path),
			Body: processed_file,
			ContentType: aws.String(media_type),
			ContentLength: aws.Int64(size),
		},
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to put object inside the s3", err)
		return
	}

	videoURL := fmt.Sprintf("https://%s/%s", cfg.s3CfDistribution, image_path)

	video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return
	}
	
	respondWithJSON(w, http.StatusOK, struct{}{})

}
