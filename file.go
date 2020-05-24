package canvas

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// FileType is a generic interface for filesystem objects
type FileType interface {
	GetID() int
	Name() string
	ParentFolder() (*Folder, error)
	Delete(...Option) error
}

// File is a file
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
	MimeClass       string      `json:"mime_class"`
	MediaEntryID    string      `json:"media_entry_id"`
	LockedForUser   bool        `json:"locked_for_user"`
	LockInfo        interface{} `json:"lock_info"`
	LockExplanation string      `json:"lock_explanation"`
	PreviewURL      string      `json:"preview_url"`

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
	resp, err := delete(f.client, fmt.Sprintf("/files/%d", f.ID), asParams(opts))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Folder is a folder
type Folder struct {
	ID         int    `json:"id"`
	Foldername string `json:"name"`
	FullName   string `json:"full_name"`

	FilesURL   string `json:"files_url"`
	FoldersURL string `json:"folders_url"`

	ContextType string `json:"context_type"`
	ContextID   int    `json:"context_id"`

	Position     int `json:"position"`
	FilesCount   int `json:"files_count"`
	FoldersCount int `json:"folders_count"`

	UpdatedAt      time.Time   `json:"updated_at"`
	LockAt         time.Time   `json:"lock_at"`
	Locked         bool        `json:"locked"`
	ParentFolderID int         `json:"parent_folder_id"`
	CreatedAt      time.Time   `json:"created_at"`
	UnlockAt       interface{} `json:"unlock_at"`
	Hidden         bool        `json:"hidden"`
	HiddenForUser  bool        `json:"hidden_for_user"`
	LockedForUser  bool        `json:"locked_for_user"`
	ForSubmissions bool        `json:"for_submissions"`

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
	return f.parent, getjson(f.client, f.parent, nil, "folders/%d", f.ParentFolderID)
}

// File gets a file by id.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_show
func (f *Folder) File(id int, opts ...Option) (*File, error) {
	file := &File{client: f.client}
	return file, getjson(f.client, file, asParams(opts), "files/%d", id)
}

// Files will return a channel that sends all of the files
// in the folder.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_index
func (f *Folder) Files() <-chan *File {
	ch := make(fileChan)
	pages := newPaginatedList(
		f.client, fmt.Sprintf("folders/%d/files", f.ID),
		sendFilesFunc(f.client, ch), nil,
	)
	go handleErrs(pages.start(), ch, ConcurrentErrorHandler)
	return ch
}

// Folders will return a channel that sends all of the sub-folders.
// https://canvas.instructure.com/doc/api/files.html#method.folders.api_index
func (f *Folder) Folders() <-chan *Folder {
	ch := make(folderChan)
	pages := newPaginatedList(
		f.client, fmt.Sprintf("folders/%d/folders", f.ID),
		sendFoldersFunc(f.client, ch), nil,
	)
	go handleErrs(pages.start(), ch, ConcurrentErrorHandler)
	return ch
}

// CreateFolder creates a new folder as a subfolder of the current one.
// https://canvas.instructure.com/doc/api/files.html#method.folders.create
func (f *Folder) CreateFolder(path string, opts ...Option) (*Folder, error) {
	dir, name := filepath.Split(path)
	return createFolder(f.client, dir, name, opts, "/folders/%d/folders", f.ID)
}

// Delete the folder
// https://canvas.instructure.com/doc/api/files.html#method.folders.api_destroy
func (f *Folder) Delete(opts ...Option) error {
	resp, err := delete(f.client, fmt.Sprintf("/folders/%d", f.ID), asParams(opts))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// https://canvas.instructure.com/doc/api/files.html#method.folders.create
func createFolder(d doer, path, name string, opts []Option, respath string, v ...interface{}) (*Folder, error) {
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
