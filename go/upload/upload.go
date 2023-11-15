package upload

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type IUpload interface {
	Run(t Setup)
	Response(data Message, w http.ResponseWriter)
}

type Upload struct {
}

type File struct {
	UploadID      string
	RequestedWith string
	CacheControl  string
	FileName      string
	SliceNum      int
	TotalSlices   int
	FileSize      int64
	SliceSize     int64
}

type Setup struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	Path       string
	Name       string
	Extensions string
	Size       int64
	Replace    bool
	Success    func(file string, idx string)
	Error      func(err error, status int)
}

type Message struct {
	File   string
	Error  string
	Status int
}

func NewUpload() IUpload {
	upload := &Upload{}

	defer func() {
		upload.close()
	}()

	return upload
}

func (u *Upload) Response(data Message, w http.ResponseWriter) {

	if data.Status == 0 {
		data.Status = http.StatusOK
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), data.Status)
		return
	}

	if data.Status != http.StatusOK {
		http.Error(w, "", data.Status)
	}

	w.Header().Set("Content-Type", "application/json")

	_, err = fmt.Fprintf(w, string(jsonData))

	if err != nil {
		http.Error(w, err.Error(), data.Status)
		return
	}
}

func (u *Upload) header(r *http.Request) (File, error) {

	sliceNumStr := r.Header.Get("X-Slice")
	sliceNum, err := strconv.Atoi(sliceNumStr)
	if err != nil {
		return File{}, err
	}

	totalSlicesStr := r.Header.Get("X-Slices")
	totalSlices, err := strconv.Atoi(totalSlicesStr)
	if err != nil {
		return File{}, err
	}

	fileSizeStr := r.Header.Get("X-File-Size")
	fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64)
	if err != nil {
		return File{}, err
	}

	sliceSizeStr := r.Header.Get("X-Slice-Size")
	sliceSize, err := strconv.ParseInt(sliceSizeStr, 10, 64)
	if err != nil {
		return File{}, err
	}

	return File{
		UploadID:    r.Header.Get("X-id"),
		FileName:    r.Header.Get("X-File-Name"),
		SliceNum:    sliceNum,
		TotalSlices: totalSlices,
		FileSize:    fileSize,
		SliceSize:   sliceSize,
	}, nil
}

func (u *Upload) checkFileSize(filesize int64, allowedSize int64) error {
	if filesize > allowedSize {
		err := fmt.Errorf("file size exceeds the allowed limit of %d bytes", allowedSize)
		return err
	}
	return nil
}

func (u *Upload) checkFileExtension(ext string, fileExtension string) error {
	r1 := regexp.MustCompile("(?i)jpg")
	r2 := regexp.MustCompile("(?i)jpeg")

	if !r1.MatchString(ext) && r2.MatchString(ext) {
		ext = fmt.Sprintf("%s %s", ext, "jpg")
	} else if r1.MatchString(ext) && !r2.MatchString(ext) {
		ext = fmt.Sprintf("%s %s", ext, "jpeg")
	}

	allowedExtensions := strings.Split(ext, " ")

	for _, extension := range allowedExtensions {
		if fmt.Sprintf(".%s", extension) == fileExtension {
			return nil
		}
	}
	err := fmt.Errorf("Invalid file extension.\n Only the following extensions are allowed:\n %s", ext)
	return err
}

func (u *Upload) directoryExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return fmt.Errorf("Failed to create directory:\n %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("Failed to check directory existence:\n %s", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("Failed to get directory information:\n %s", err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	return nil
}

func (u *Upload) removeFile(path string, name string) error {
	file := filepath.Join(path, name)

	if _, err := os.Stat(file); err == nil {
		err = os.Remove(file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *Upload) isDirectoryWritable(path string) error {

	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not direct", path)
	}

	fileMode := fileInfo.Mode()
	if fileMode.Perm()&(1<<uint(7)) == 0 {
		err := os.Chmod(path, 0644)

		if err != nil {
			return fmt.Errorf("failed to set directory %s writable: %s", path, err.Error())
		}
	}

	return nil
}

func (u *Upload) openUploadFile(filePath string) (file *os.File, err error) {
	file, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)

	if os.IsNotExist(err) {
		file, err = os.Create(filePath)
	}

	if err != nil {
		log.Printf("Failed to create %s: %s", filePath, err)
	}

	return file, err
}

func (u *Upload) closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		fmt.Println(err)
	}
}

func (u *Upload) Run(t Setup) {

	var err error

	if err = t.Request.ParseMultipartForm(32 << 20); err != nil {
		t.Error(err, http.StatusBadRequest)
		return
	}

	err = u.directoryExists(t.Path)

	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	err = u.isDirectoryWritable(t.Path)

	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	fileInfo, err := u.header(t.Request)

	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	err = u.checkFileSize(fileInfo.FileSize, t.Size)

	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	extension := filepath.Ext(fileInfo.FileName)

	err = u.checkFileExtension(t.Extensions, extension)
	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	data, _, err := t.Request.FormFile("file")
	if err != nil {
		t.Error(errors.New("failed to retrieve file"), http.StatusInternalServerError)
		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			t.Error(errors.New("failed to retrieve file"), http.StatusInternalServerError)
		}
	}(data)

	var filePath string

	if t.Name == "" {
		filePath = fmt.Sprintf("%s%s", t.Path, fileInfo.FileName)
	} else {
		filePath = fmt.Sprintf("%s%s%s", t.Path, t.Name, extension)
	}

	file, err := u.openUploadFile(filePath)
	defer u.closeFile(file)

	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	chunkSize := int64(4096)

	err = u.uploadDataChunksToFile(file, data, chunkSize)
	if err != nil {
		t.Error(errors.New("failed to upload data"), http.StatusInternalServerError)
		return
	}

	if fileInfo.TotalSlices == fileInfo.SliceNum {
		newFilePath := strings.TrimPrefix(filePath, ".")
		t.Success(newFilePath, fileInfo.UploadID)
	}
}

func (u *Upload) uploadDataChunksToFile(file *os.File, data io.Reader, chunkSize int64) error {
	buffer := make([]byte, chunkSize)
	for {
		n, err := data.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		_, err = file.Write(buffer[:n])
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *Upload) close() {
	*u = Upload{}
}
