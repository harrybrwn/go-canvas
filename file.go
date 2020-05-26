package canvas

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// FileType is a generic interface for filesystem objects
type FileType interface {
	GetID() int
	Name() string
	ParentFolder() (*Folder, error)
	Delete(...Option) error
}

// File is a file.
// https://canvas.instructure.com/doc/api/files.html
type File struct {
	ID       int    `json:"id"`
	FolderID int    `json:"folder_id"`
	URL      string `json:"url"`
	UUID     string `json:"uuid"`

	Filename    string `json:"filename"`
	DisplayName string `json:"display_name"`

	ContentType string    `json:"content-type"`
	Size        int       `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ModifiedAt  time.Time `json:"modified_at"`

	Locked   bool      `json:"locked"`
	UnlockAt time.Time `json:"unlock_at"`
	Hidden   bool      `json:"hidden"`
	LockAt   time.Time `json:"lock_at"`

	HiddenForUser   bool        `json:"hidden_for_user"`
	ThumbnailURL    string      `json:"thumbnail_url"`
	PreviewURL      string      `json:"preview_url"`
	MimeClass       string      `json:"mime_class"`
	MediaEntryID    string      `json:"media_entry_id"`
	LockedForUser   bool        `json:"locked_for_user"`
	LockInfo        interface{} `json:"lock_info"`
	LockExplanation string      `json:"lock_explanation"`

	client doer
	folder *Folder
}

// Name returns the file's filename
func (f *File) Name() string {
	return f.Filename
}

// GetID is for the FileType interface
func (f *File) GetID() int {
	return f.ID
}

// ParentFolder will get the folder that the file is a part of.
func (f *File) ParentFolder() (*Folder, error) {
	if f.folder != nil && f.folder.ID == f.FolderID {
		return f.folder, nil
	}
	f.folder = &Folder{client: f.client}
	err := getjson(f.client, f.folder, nil, "folders/%d", f.FolderID)
	return f.folder, err
}

// PublicURL will get the file's public url.
func (f *File) PublicURL() (string, error) {
	resp, err := get(f.client, fmt.Sprintf("/files/%d/public_url", f.ID), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	m := make(map[string]interface{})
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", err
	}
	u, ok := m["public_url"]
	if !ok {
		return "", errors.New("could not find public url")
	}
	return u.(string), nil
}

// Delete the file.
// https://canvas.instructure.com/doc/api/files.html#method.files.destroy
func (f *File) Delete(opts ...Option) error {
	resp, err := delete(
		f.client,
		fmt.Sprintf("/files/%d", f.ID),
		optEnc(opts),
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Move a file to another folder.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_update
func (f *File) Move(folder *Folder, opts ...Option) error {
	return f.edit(append(opts, Opt("parent_folder_id", folder.ID)))
}

// Rename the file.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_update
func (f *File) Rename(name string, opts ...Option) error {
	return f.edit(append(opts, Opt("name", name)))
}

func (f *File) edit(opts optEnc) error {
	resp, err := put(
		f.client,
		fmt.Sprintf("/files/%d", f.ID),
		opts,
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Folder is a folder
type Folder struct {
	ID             int    `json:"id"`
	ParentFolderID int    `json:"parent_folder_id"`
	Foldername     string `json:"name"`
	FullName       string `json:"full_name"`

	FilesURL   string `json:"files_url"`
	FoldersURL string `json:"folders_url"`

	ContextType string `json:"context_type"`
	ContextID   int    `json:"context_id"`

	Position     int `json:"position"`
	FilesCount   int `json:"files_count"`
	FoldersCount int `json:"folders_count"`

	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LockAt         time.Time `json:"lock_at"`
	UnlockAt       time.Time `json:"unlock_at"`
	Locked         bool      `json:"locked"`
	Hidden         bool      `json:"hidden"`
	HiddenForUser  bool      `json:"hidden_for_user"`
	LockedForUser  bool      `json:"locked_for_user"`
	ForSubmissions bool      `json:"for_submissions"`

	client doer
	parent *Folder
}

// Name returns the folder's name
func (f *Folder) Name() string {
	return f.Foldername
}

// GetID will return the folder id, this is only here for interfaces.
func (f *Folder) GetID() int {
	return f.ID
}

// ParentFolder will get the folder's parent folder.
func (f *Folder) ParentFolder() (*Folder, error) {
	if f.parent != nil {
		return f.parent, nil
	}
	f.parent = &Folder{client: f.client}
	return f.parent, getjson(
		f.client, f.parent, nil,
		"folders/%d", f.ParentFolderID,
	)
}

// File gets a file by id.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_show
func (f *Folder) File(id int, opts ...Option) (*File, error) {
	file := &File{client: f.client}
	return file, getjson(f.client, file, optEnc(opts), "files/%d", id)
}

// Files will return a channel that sends all of the files
// in the folder.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_index
func (f *Folder) Files(opts ...Option) <-chan *File {
	return filesChannel(
		f.client, fmt.Sprintf("folders/%d/files", f.ID),
		ConcurrentErrorHandler, opts,
	)
}

// Folders will return a channel that sends all of the sub-folders.
// https://canvas.instructure.com/doc/api/files.html#method.folders.api_index
func (f *Folder) Folders() <-chan *Folder {
	ch := make(folderChan)
	pages := newPaginatedList(
		f.client,
		fmt.Sprintf("folders/%d/folders", f.ID),
		sendFoldersFunc(f.client, ch), nil,
	)
	go handleErrs(pages.start(), ch, ConcurrentErrorHandler)
	return ch
}

// CreateFolder creates a new folder as a subfolder of the current one.
// https://canvas.instructure.com/doc/api/files.html#method.folders.create
func (f *Folder) CreateFolder(path string, opts ...Option) (*Folder, error) {
	dir, name := filepath.Split(path)
	return createFolder(
		f.client, dir,
		name, opts,
		"/folders/%d/folders", f.ID,
	)
}

// Delete the folder
// https://canvas.instructure.com/doc/api/files.html#method.folders.api_destroy
func (f *Folder) Delete(opts ...Option) error {
	resp, err := delete(
		f.client,
		fmt.Sprintf("/folders/%d", f.ID),
		optEnc(opts),
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// UploadFile uploads a file into a specific file.
func (f *Folder) UploadFile(
	filename string,
	r io.Reader,
	opts ...Option,
) (*File, error) {
	opts = append(opts, Opt("parent_folder_id", f.ID))
	path := fmt.Sprintf("/folders/%d/files", f.ID)
	return uploadFile(f.client, filename, r, path, opts)
}

func filesChannel(
	d doer,
	path string,
	handler errorHandlerFunc,
	opts []Option,
) <-chan *File {
	ch := make(fileChan)
	pager := newPaginatedList(d, path, sendFilesFunc(d, ch), opts)
	go handleErrs(pager.start(), ch, handler)
	return ch
}

func uploadFile(
	d doer,
	filename string,
	r io.Reader,
	path string,
	opts []Option,
) (*File, error) {
	q := params{"name": {filename}}
	q.Add(opts...)
	query := q.Encode()

	req := newreq("POST", path, query)
	resp, err := do(d, req)
	if err != nil {
		return nil, err
	}
	uploader, err := getUploader(resp.Body)
	if err != nil {
		return nil, err
	}
	// req, _ = http.NewRequest("OPTIONS", uploader.UploadURL, nil)
	// resp, err = d.Do(req)
	// if err != nil {
	// 	panic(err)
	// }
	// b, _ := ioutil.ReadAll(resp.Body)
	// fmt.Printf("%s\n", b)
	// fmt.Println(resp.Status)
	return uploader.upload(d, filename, r)
}

// https://canvas.instructure.com/doc/api/files.html#method.folders.create
func createFolder(
	d doer,
	path, name string,
	opts []Option,
	respath string,
	v ...interface{},
) (*Folder, error) {
	parentpath, name := filepath.Split(path)
	q := params{"name": {name}}
	if parentpath != "" {
		q["parent_folder_path"] = []string{parentpath}
	}
	for _, o := range opts {
		if _, ok := q[o.Name()]; !ok {
			q[o.Name()] = o.Value()
		}
	}

	resp, err := post(d, fmt.Sprintf(respath, v...), q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	f := &Folder{client: d}
	return f, json.NewDecoder(resp.Body).Decode(f)
}

func getUploader(rc io.ReadCloser) (*fileupload, error) {
	defer rc.Close()
	fup := &fileupload{}
	return fup, json.NewDecoder(rc).Decode(fup)
}

type fileupload struct {
	FileParam    string       `json:"file_param"`
	Progress     string       `json:"progress"`
	UploadURL    string       `json:"upload_url"`
	UploadParams genericParam `json:"upload_params"`
}

func (f *fileupload) upload(d doer, filename string, r io.Reader) (*File, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range f.UploadParams {
		writer.WriteField(key, fmt.Sprintf("%v", value))
	}

	form, err := writer.CreateFormFile(f.FileParam, filename)
	if err != nil {
		return nil, err
	}

	if _, err = io.Copy(form, r); err != nil {
		return nil, err
	}
	writer.Close()

	u, err := url.Parse(f.UploadURL)
	if err != nil {
		return nil, err
	}
	req := &http.Request{
		Method: "POST",
		URL:    u,
		Body:   ioutil.NopCloser(&body),
		Header: http.Header{
			"Content-Type": {writer.FormDataContentType()}},
		ContentLength: int64(body.Len()),
	}
	fmt.Println(body.String())
	// resp, err := do(d, req)
	resp, err := d.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fmt.Printf("%+v\n", resp.Status)
	file := &File{client: d}
	return file, json.NewDecoder(resp.Body).Decode(file)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func folderList(d doer, path string) ([]*Folder, error) {
	folders := []*Folder{}
	err := getjson(d, &folders, nil, path)
	if err != nil {
		return nil, err
	}
	for i := range folders {
		folders[i].client = d
	}
	return folders, nil
}

var (
	_ FileType = (*File)(nil)
	_ FileType = (*Folder)(nil)
)

type fileChan chan *File

func (fc fileChan) Close() {
	close(fc)
}

type folderChan chan *Folder

func (fc folderChan) Close() {
	close(fc)
}
