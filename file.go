package canvas

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// File is a file
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

// ParentFolder will get the folder that the file is a part of.
func (f *File) ParentFolder() (*Folder, error) {
	if f.folder != nil && f.folder.ID == f.FolderID {
		return f.folder, nil
	}
	f.folder = &Folder{client: f.client}
	err := getjson(f.client, f.folder, nil, "folders/%d", f.FolderID)
	return f.folder, err
}

// Folder is a folder
type Folder struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`

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

// ParentFolder will get the folder's parent folder.
func (f *Folder) ParentFolder() (*Folder, error) {
	if f.parent != nil {
		return f.parent, nil
	}
	f.parent = &Folder{client: f.client}
	return f.parent, getjson(f.client, f.parent, nil, "folders/%d", f.ParentFolderID)
}

// Files will return a channel that sends all of the files
// in the folder.
func (f *Folder) Files() <-chan *File {
	pages := newPaginatedList(
		f.client,
		fmt.Sprintf("folders/%d/files", f.ID),
		filesInitFunc(f.client),
		nil,
	)
	return onlyFiles(pages, ConcurrentErrorHandler)
}

// Folders will return a channel that sends all of the sub-folders.
func (f *Folder) Folders() <-chan *Folder {
	pages := newPaginatedList(
		f.client,
		fmt.Sprintf("folders/%d/folders", f.ID),
		filesInitFunc(f.client),
		nil,
	)
	return onlyFolders(pages, ConcurrentErrorHandler)
}

func filesInitFunc(c doer) pageInitFunction {
	return func(page int, r io.Reader) ([]interface{}, error) {
		files := make([]*File, 0)
		if err := json.NewDecoder(r).Decode(&files); err != nil {
			return nil, err
		}
		objects := make([]interface{}, len(files))
		for i, f := range files {
			f.client = c
			objects[i] = f
		}
		return objects, nil
	}
}

func foldersInitFunc(c doer) pageInitFunction {
	return func(page int, r io.Reader) ([]interface{}, error) {
		folders := make([]*Folder, 0)
		if err := json.NewDecoder(r).Decode(&folders); err != nil {
			return nil, err
		}
		objects := make([]interface{}, len(folders))
		for i, f := range folders {
			f.client = c
			objects[i] = f
		}
		return objects, nil
	}
}
