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

// FileObjType is the type of a file
type FileObjType int

const (
	// TypeFile is the type for files
	TypeFile FileObjType = iota
	// TypeFolder is the type for folders
	TypeFolder
)

// FileObj is a interface for filesystem objects
type FileObj interface {
	GetID() int
	Type() FileObjType
	Name() string
	Path() string

	Move(*Folder) error
	Rename(string) error
	Copy(*Folder) error
	Delete(...Option) error
	Hide() error
	Unhide() error

	ParentFolder() (*Folder, error)
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
	return f.DisplayName
}

// Type returns canvas.TypeFile
func (f *File) Type() FileObjType {
	return TypeFile
}

// Path returns the folder path that the file is in.
func (f *File) Path() string {
	fldr, _ := f.ParentFolder()
	if fldr == nil {
		return ""
	}
	return fldr.FullName
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

// Copy the file into another folder.
// https://canvas.instructure.com/doc/api/files.html#method.folders.copy_file
func (f *File) Copy(dest *Folder) error {
	resp, err := post(
		f.client,
		fmt.Sprintf("/folders/%d/copy_file", dest.ID),
		params{
			"source_file_id": {f.strID()},
			"on_duplicate":   {"rename"},
		},
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Move a file to another folder.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_update
func (f *File) Move(dest *Folder) error {
	id := dest.GetID()
	if id <= 0 && dest.FullName != "" {
		return f.edit(Opt("parent_folder_path", dest.FullName))
	}
	return f.edit(Opt("parent_folder_id", id))
}

// Rename the file.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_update
func (f *File) Rename(name string) error {
	return f.edit(Opt("name", name))
}

// Hide the file
func (f *File) Hide() error {
	return f.edit(Opt("hidden", true))
}

// Unhide the file
func (f *File) Unhide() error {
	return f.edit(Opt("hidden", false))
}

func (f *File) edit(opts ...Option) error {
	resp, err := put(
		f.client,
		fmt.Sprintf("/files/%d", f.ID),
		optEnc(opts),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(f)
}

// WriteTo will write the contents of the file to an io.Writer
func (f *File) WriteTo(w io.Writer) (int64, error) {
	resp, err := http.Get(f.URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return io.Copy(w, resp.Body)
}

func (f *File) strID() string {
	return fmt.Sprintf("%d", f.ID)
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

// Name returns only the folder's name without the path.
func (f *Folder) Name() string {
	return f.Foldername
}

// Path wil return only the folder's path without it's name.
func (f *Folder) Path() string {
	dir, _ := filepath.Split(f.FullName)
	return dir
}

// GetID will return the folder id, this is only here for interfaces.
func (f *Folder) GetID() int {
	return f.ID
}

// Type returns canvas.TypeFolder
func (f *Folder) Type() FileObjType {
	return TypeFolder
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
		ConcurrentErrorHandler, opts, f,
	)
}

// Folders will return a channel that sends all of the sub-folders.
// https://canvas.instructure.com/doc/api/files.html#method.folders.api_index
func (f *Folder) Folders() <-chan *Folder {
	ch := make(folderChan)
	pages := newPaginatedList(
		f.client,
		fmt.Sprintf("folders/%d/folders", f.ID),
		sendFoldersFunc(f.client, ch, f),
		nil,
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

// Copy the folder to a another folder (dest)
// https://canvas.instructure.com/doc/api/files.html#method.folders.copy_folder
func (f *Folder) Copy(dest *Folder) error {
	resp, err := post(
		f.client,
		fmt.Sprintf("/folders/%d/copy_folder", dest.ID),
		params{"source_folder_id": {fmt.Sprintf("%d", f.ID)}},
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Rename the folder.
func (f *Folder) Rename(name string) error {
	return f.edit(Opt("name", name))
}

// Move the folder into another folder
func (f *Folder) Move(dest *Folder) error {
	id := dest.GetID()
	if id <= 0 && dest.FullName != "" {
		return f.edit(Opt("parent_folder_path", dest.FullName))
	}
	return f.edit(Opt("parent_folder_id", id))
}

// Hide the folder
func (f *Folder) Hide() error {
	return f.edit(Opt("hidden", true))
}

// Unhide the folder
func (f *Folder) Unhide() error {
	return f.edit(Opt("hidden", false))
}

// Delete the folder
// https://canvas.instructure.com/doc/api/files.html#method.folders.api_destroy
func (f *Folder) Delete(opts ...Option) error {
	resp, err := delete(
		f.client, fmt.Sprintf("/folders/%d", f.ID),
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

// https://canvas.instructure.com/doc/api/files.html#method.folders.update
func (f *Folder) edit(opts ...Option) error {
	resp, err := put(f.client, fmt.Sprintf("/folders/%d", f.ID), optEnc(opts))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(f)
}

func filesChannel(
	d doer,
	path string,
	handler errorHandlerFunc,
	opts []Option,
	parent *Folder,
) <-chan *File {
	ch := make(fileChan)
	pager := newPaginatedList(d, path, sendFilesFunc(d, ch, parent), opts)
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
	q.Add(opts)

	req := newreq("POST", path, q.Encode())
	resp, err := do(d, req)
	if err != nil {
		return nil, err
	}
	uploader, err := getUploader(resp.Body) // will close the body
	if err != nil {
		return nil, err
	}
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
		q.Set("parent_folder_path", parentpath)
	}
	q.Add(opts)

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
	b := &bytes.Buffer{}
	fup := &fileupload{
		body:   b,
		writer: multipart.NewWriter(b),
	}
	err := json.NewDecoder(rc).Decode(fup)
	if err != nil {
		return nil, err
	}
	for key, value := range fup.UploadParams {
		fup.writer.WriteField(key, fmt.Sprintf("%v", value))
	}
	fup.url, err = url.Parse(fup.UploadURL)
	if err != nil {
		return nil, err
	}
	return fup, nil
}

type fileupload struct {
	FileParam    string       `json:"file_param"`
	Progress     string       `json:"progress"`
	UploadURL    string       `json:"upload_url"`
	UploadParams genericParam `json:"upload_params"`

	url    *url.URL
	body   *bytes.Buffer
	writer *multipart.Writer
}

func (f *fileupload) upload(d doer, filename string, r io.Reader) (*File, error) {
	form, err := f.writer.CreateFormFile(f.FileParam, filename)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(form, r); err != nil {
		return nil, err
	}
	f.writer.Close() // do not defer, adds the correct line endings to the body
	req := &http.Request{
		Method: "POST",
		URL:    f.url,
		Body:   ioutil.NopCloser(f.body),
		Header: http.Header{
			"Content-Type": {f.writer.FormDataContentType()}},
		ContentLength: int64(f.body.Len()),
	}
	resp, err := do(d, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
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
	_ FileObj     = (*File)(nil)
	_ io.WriterTo = (*File)(nil)
	_ FileObj     = (*Folder)(nil)
)

type fileChan chan *File

func (fc fileChan) Close() {
	close(fc)
}

type folderChan chan *Folder

func (fc folderChan) Close() {
	close(fc)
}

func (f *File) setclient(d doer) {
	f.client = d
}

func (f *Folder) setclient(d doer) {
	f.client = d
}
