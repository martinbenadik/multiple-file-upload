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
	Parameter(r *http.Request) string
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
	Parameter     string
}

type Setup struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	Path       string
	Name       string
	Sub        string
	Normalize  bool
	Extensions string
	Size       int64
	Replace    bool
	Success    func(u SuccessObject)
	Error      func(err error, status int)
}

type Message struct {
	File      string
	Parameter string
	Name      string
	Path      string
	Id        string
	Error     string
	Status    int
}

func NewUpload() IUpload {
	upload := &Upload{}

	defer func() {
		upload.close()
	}()

	return upload
}

func (u *Upload) Parameter(r *http.Request) string {
	parameter := r.Header.Get("X-Parameter")
	return parameter
}

func (u *Upload) getName(path string, name string, t Setup) string {
	fn := fmt.Sprintf("%s-?", t.Name)
	extensions := strings.Join(strings.Split(t.Extensions, " "), "|")

	re := regexp.MustCompile(`\?`)
	fileNamePattern := re.ReplaceAllStringFunc(fn, func(match string) string {
		cl := fmt.Sprintf("(\\d+)\\.(%s)", extensions)
		return fmt.Sprintf("%s", cl)
	})

	pattern := regexp.MustCompile(fileNamePattern)
	files, _ := os.ReadDir(path)

	m := 0
	for _, file := range files {
		_fileName_ := file.Name()
		match := pattern.FindStringSubmatch(_fileName_)
		if len(match) == 3 {
			i, err := strconv.Atoi(match[1])
			if err != nil {
				return name
			}
			if m <= i {
				m = i
			}
		}
	}
	m++

	re = regexp.MustCompile(`\?`)
	fn = re.ReplaceAllStringFunc(fn, func(match string) string {
		cl := m
		return fmt.Sprintf("%d", cl)
	})

	return fmt.Sprintf("%s%s", fn, filepath.Ext(name))
}

func (u *Upload) save(sourcePath string, destinationPath string) error {

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(sourcePath, destinationPath); err != nil {
		return err
	}

	return nil
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

	parameter := r.Header.Get("X-Parameter")

	unique := r.Header.Get("X-Unique")

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

	fileName := fmt.Sprintf("__%s·%s", unique, r.Header.Get("X-File-Name"))

	idx := r.Header.Get("X-id")

	if idx == "null" {
		idx = ""
	}

	return File{
		UploadID:    idx,
		FileName:    fileName,
		SliceNum:    sliceNum,
		TotalSlices: totalSlices,
		FileSize:    fileSize,
		SliceSize:   sliceSize,
		Parameter:   parameter,
	}, nil
}

func (u *Upload) checkFileSize(filesize int64, allowedSize int64) error {
	if filesize > allowedSize {
		err := fmt.Errorf("file size exceeds the allowed limit of %d bytes", allowedSize)
		return err
	}
	return nil
}

func (u *Upload) checkFileExtension(ext string, fileExtension string) (string, error) {
	ext = strings.ToLower(ext)
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
			return ext, nil
		}
	}
	err := fmt.Errorf("Invalid file extension.\n Only the following extensions are allowed:\n %s", ext)
	return ext, err
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

func (u *Upload) removeFile(filePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		err = os.Remove(filePath)
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

type SuccessObject struct {
	Id        string
	File      string
	Name      string
	Path      string
	Parameter string
}

type ErrorObject struct {
	FilePath string
	Error    error
	Status   int
	Fn       func(err error, status int)
}

func (u *Upload) errorMiddleware(obj ErrorObject) {

	if obj.FilePath != "" {
		_ = u.removeFile(obj.FilePath)
	}

	obj.Fn(obj.Error, obj.Status)
}

func (u *Upload) fixExtension(filename string) string {
	regex := regexp.MustCompile("\\.(jpeg|JPEG|JPG)$")
	return regex.ReplaceAllString(filename, ".jpg")
}

func (u *Upload) Run(t Setup) {

	var err error

	if err = t.Request.ParseMultipartForm(32 << 20); err != nil {
		u.errorMiddleware(ErrorObject{
			FilePath: "",
			Error:    errors.New("problem parse multipart form"),
			Status:   http.StatusBadRequest,
			Fn:       t.Error,
		})
		return
	}
	if t.Path[len(t.Path)-1:] != "/" {
		t.Path = t.Path + "/"
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

	if t.Normalize {
		fileInfo.FileName = u.fixExtension(fileInfo.FileName)
	}

	if err != nil {
		t.Error(errors.New("can't get file data from the header"), http.StatusInternalServerError)
		return
	}

	err = u.checkFileSize(fileInfo.FileSize, t.Size)

	if err != nil {
		t.Error(errors.New("file size problem"), http.StatusInternalServerError)
		return
	}

	extension := strings.ToLower(filepath.Ext(fileInfo.FileName))
	t.Extensions, err = u.checkFileExtension(t.Extensions, extension)
	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	data, _, err := t.Request.FormFile("file")
	if err != nil {
		t.Error(errors.New("failed to retrieve a file"), http.StatusInternalServerError)
		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			t.Error(errors.New("failed to retrieve a file"), http.StatusInternalServerError)
			return
		}
	}(data)

	var filePath string

	filePath = fmt.Sprintf("%s%s", t.Path, fileInfo.FileName)

	file, err := u.openUploadFile(filePath)
	defer u.closeFile(file)

	if err != nil {
		t.Error(err, http.StatusInternalServerError)
		return
	}

	chunkSize := int64(4096)

	err = u.uploadDataChunksToFile(file, data, chunkSize)
	if err != nil {
		u.errorMiddleware(ErrorObject{
			FilePath: "",
			Error:    errors.New("failed to upload data chunks to the file"),
			Status:   http.StatusInternalServerError,
			Fn:       t.Error,
		})
		return
	}

	if fileInfo.TotalSlices == fileInfo.SliceNum {
		var (
			newFilePath string
			newFileName string
		)

		filePath := t.Path
		fileName := fileInfo.FileName
		sourcePath := fmt.Sprintf("%s%s", filePath, fileName)
		f := strings.Split(fileInfo.FileName, "·")
		newFileName = f[1]

		if t.Sub != "" {
			newFilePath = fmt.Sprintf("%s%s/", filePath, t.Sub)
			err = u.directoryExists(newFilePath)
			if err != nil {
				u.errorMiddleware(ErrorObject{
					FilePath: sourcePath,
					Error:    err,
					Status:   http.StatusInternalServerError,
					Fn:       t.Error,
				})
			}
		} else {
			newFilePath = filePath
		}

		if t.Name != "" {
			newFileName = u.getName(newFilePath, newFileName, t)
		}

		destinationPath := fmt.Sprintf("%s%s", newFilePath, newFileName)

		err := u.save(sourcePath, destinationPath)
		if err != nil {
			u.errorMiddleware(ErrorObject{
				FilePath: sourcePath,
				Error:    err,
				Status:   http.StatusInternalServerError,
				Fn:       t.Error,
			})
		}

		t.Success(SuccessObject{
			Id:        fileInfo.UploadID,
			File:      strings.TrimPrefix(destinationPath, "."),
			Name:      newFileName,
			Path:      strings.TrimPrefix(newFilePath, "."),
			Parameter: fileInfo.Parameter,
		})

		return
	}
}

func (u *Upload) rename(filePath string, newFilePath string) (string, error) {

	err := os.Rename(filePath, newFilePath)
	if err != nil {
		return filePath, err
	}

	return newFilePath, nil
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
