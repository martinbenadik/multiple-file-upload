package main

import (
	"log"
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
		Size:       1024 * 1024 * 32,
		Success: func(file string, idx string) {
			Upload.Response(upload.Message{
				File: file,
			}, w)
		},
		Error: func(err error, httpErrorStatus int) {

			log.Printf("Error: %v", err)

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

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		return
	}
}
