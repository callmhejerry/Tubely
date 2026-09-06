package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIdString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIdString)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video id", err)
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

	fmt.Println("uploading video", videoID, "by user", userID)

	maxUploadSize := 1 << 30

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxUploadSize))

	video, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusNotFound, "Not found", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	videoFile, videoFileHeader, err := r.FormFile("video")

	defer videoFile.Close()

	mediaType, _, err := mime.ParseMediaType(videoFileHeader.Header.Get("Content-Type"))

	fileExt := strings.Split(mediaType, "/")[1]

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse Content-Type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "You can only upload mp4 files", err)
		return
	}

	tempVideoFile, err := os.CreateTemp("", "tubely-upload.mp4")

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create temporary video file", err)
		return
	}

	defer os.Remove(tempVideoFile.Name())
	defer tempVideoFile.Close()

	if _, err := io.Copy(tempVideoFile, videoFile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to copy video file to temporary location", err)
		return
	}

	tempVideoFile.Seek(0, io.SeekStart)

	aspectRatio, err := getVideoAspectRatio(tempVideoFile.Name())

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get aspect ratio of file", err)
		return
	}

	preprocessedVideoPath, err := ProcessVideoForFastStart(tempVideoFile.Name())

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	processedVideo, err := os.Open(preprocessedVideoPath)
	defer processedVideo.Close()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to open processed video", err)
		return
	}

	buf := make([]byte, 32)
	rand.Read(buf)

	fileKey := fmt.Sprintf("%s/%s.%s", aspectRatio, base64.RawURLEncoding.EncodeToString(buf), fileExt)

	cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        processedVideo,
		ContentType: &mediaType,
	})

	videoFileUrl := fmt.Sprintf("%s,%s", cfg.s3Bucket, fileKey)

	video.VideoURL = &videoFileUrl
	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}
	videoWithPresignedUrl, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoWithPresignedUrl)
}
