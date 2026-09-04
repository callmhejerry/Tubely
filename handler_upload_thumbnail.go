package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	maxMemory := 2 << 10
	r.ParseMultipartForm(int64(maxMemory))
	thumbnailFile, thumbnailHeader, err := r.FormFile("thumbnail")

	mediaType, _, err := mime.ParseMediaType(thumbnailHeader.Header.Get("Content-Type"))

	fmt.Printf("MediaType: %s\n", mediaType)

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type for thumbnail", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 400, "Something went wrong", err)
		return
	}
	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	thumbnailFileExtension := strings.Split(mediaType, "/")[1]

	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	thumbnailFileName := base64.RawURLEncoding.EncodeToString(randomBytes)

	thumbnailFilePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", thumbnailFileName, thumbnailFileExtension))

	newThumbnailFile, err := os.Create(thumbnailFilePath)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %s", err.Error(), thumbnailFilePath), err)
		return
	}

	if _, err := io.Copy(newThumbnailFile, thumbnailFile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to copy thumbnail file to new destination", err)
		return
	}

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, thumbnailFileName, thumbnailFileExtension)
	video.ThumbnailURL = &thumbnailUrl

	cfg.db.UpdateVideo(video)
	respondWithJSON(w, http.StatusOK, video)
}
