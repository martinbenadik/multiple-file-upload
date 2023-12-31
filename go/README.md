# GoLang Multiple File Upload

## Example Usage

Here's a simple example demonstrating the usage of the package:

```
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
```