package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	max_bytes := 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, int64(max_bytes))
	videoId, err := uuid.Parse(r.PathValue("videoID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "videoid parse error", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldnt find jwt", err)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldnt validate jwt", err)
		return
	}
	video_metadata, err := cfg.db.GetVideo(videoId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "db error", err)
		return
	}
	if video_metadata.UserID != userId {
		respondWithError(w, http.StatusUnauthorized, "userid didnt match", errors.New("userid didnt match"))
		return
	}

	err = r.ParseMultipartForm(int64(max_bytes))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "multipart parse error", err)
		return
	}
	file_data, file_header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting formfile", err)
		return
	}
	defer file_data.Close()
	content_type := file_header.Header.Get("Content-Type")
	if content_type != "video/mp4" {
		respondWithError(w, http.StatusInternalServerError, "wrong file type", errors.New("wrong file type"))
		return
	}
	temp_file, err := os.CreateTemp("", "tubely_temp_upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusTemporaryRedirect, "temp file createion error", err)
		return
	}
	defer os.Remove(temp_file.Name())
	defer temp_file.Close()
	_, err = io.Copy(temp_file, file_data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "file write error", err)
		return
	}
	_, err = temp_file.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "file seek erro", err)
		return
	}

	key_bytes_slice := make([]byte, 32)
	_, err = rand.Read(key_bytes_slice)
	key_val := hex.EncodeToString(key_bytes_slice)
	file_key := fmt.Sprintf("%s.mp4", key_val)
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &file_key,
		Body:        temp_file,
		ContentType: &content_type,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "aws s3 put object error", err)
		return
	}
	video_url := fmt.Sprintf(
		"https://%s.s3.%s.amazonaws.com/%s",
		cfg.s3Bucket,
		cfg.s3Region,
		file_key,
	)
	video_metadata.VideoURL = &video_url
	err = cfg.db.UpdateVideo(video_metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "db error", err)
		return
	}

}
