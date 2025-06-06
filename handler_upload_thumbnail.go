package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

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
	if content_type != "image/jpeg" {
		if content_type != "image/png" {
			respondWithError(w, http.StatusInternalServerError, "file type error", errors.New("File type error"))
			return
		}
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
	file_extension, err := mime.ExtensionsByType(content_type)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading extension", err)
		return
	}
	file_name_slice := make([]byte, 32)
	_, err = rand.Read(file_name_slice)
	file_name := base64.RawURLEncoding.EncodeToString(file_name_slice)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Rand gen error", err)
		return
	}
	file_path := fmt.Sprintf("%s/%s%s", cfg.assetsRoot, file_name, file_extension[0])
	file, err := os.Create(file_path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "file creation error", err)
		return
	}
	_, err = io.Copy(file, file_data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "file write error", err)
		return
	}
	thumbnail_url := fmt.Sprintf("http://localhost:8091/assets/%s%s", file_name, file_extension[0])
	video.ThumbnailURL = &thumbnail_url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "db error", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
