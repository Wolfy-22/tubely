package main

import (
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

const maxMemory = 10 << 20

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

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing thumbnail", err)
		return
	}

	thumbnailData, thumbanilHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error forming thumbnail", err)
		return
	}

	contentType := thumbanilHeader.Header.Get("Content-Type")

	_, err = io.ReadAll(thumbnailData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading bytes", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video Does not Exist", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not owner of video", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing media type", err)
		return
	}

	ext := strings.Split(mediaType, "/")

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusNotAcceptable, "content type must be either image/jpeg or image/png", err)
		return
	}

	filePath := filepath.Join(cfg.assetsRoot, videoIDString)

	file, err := os.Create(fmt.Sprintf("%v.%v", filePath, ext[1]))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating file", err)
		return
	}

	_, err = io.Copy(file, thumbnailData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving thumbnail", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:8091/%v", filePath)

	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	respondWithJSON(w, http.StatusOK, video)
}
