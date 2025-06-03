package main

import (
	"errors"
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

	maxMemory := 10 << 20
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Multipart Form Parse error", err)
		return
	}
	file_data, file_header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error with FormFile", err)
		return
	}
	content_type := file_header.Header.Get("Content-Type")
	image_data_bytes, err := io.ReadAll(file_data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "read error", err)
		return
	}
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "db error", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "userid doesent match", errors.New("UserId doesent match"))
		return
	}
	added_thumbnail := thumbnail{data: image_data_bytes, mediaType: content_type}
	videoThumbnails[videoID] = added_thumbnail
	thumbnail_url := fmt.Sprintf("http://localhost:8091/api/thumbnails/%v", videoID)
	video.ThumbnailURL = &thumbnail_url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "db error", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
