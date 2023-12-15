package main

import (
	"multipleFileUpload/upload"
	"net/http"
)

func fileUpload(w http.ResponseWriter, r *http.Request) {

	Upload := upload.NewUpload()

	Upload.Run(upload.Setup{
		Writer:     w,
		Request:    r,
		Path:       "./static/images/",
		Extensions: "gif jpg png webp",
		Name:       "image",
		Sub:        Upload.Parameter(r),
		Normalize:  true,
		Size:       1024 * 1024 * 32,
		Success: func(u upload.SuccessObject) {
			Upload.Response(upload.Message{
				Id:        u.Id,
				File:      u.File,
				Path:      u.Path,
				Name:      u.Name,
				Parameter: u.Parameter,
			}, w)
		},
		Error: func(err error, httpErrorStatus int) {
			Upload.Response(upload.Message{
				Error:  err.Error(),
				Status: httpErrorStatus,
			}, w)
		},
	})
}

func main() {

	http.HandleFunc("/upload", fileUpload)

	fs := http.FileServer(http.Dir("./static/images"))

	http.Handle("/static/images/", http.StripPrefix("/static/images/", fs))

	err := http.ListenAndServe("localhost:8000", nil)
	if err != nil {
		return
	}
}
