package canvas

import (
	"net/url"
	"path/filepath"
)

type params map[string][]string

type encoder interface {
	Encode() string
}

func (p params) Add(vals []Option) {
	for _, v := range vals {
		p[v.Name()] = v.Value()
	}
}

func (p params) Set(key, val string) {
	p[key] = []string{val}
}

// Encode converts the params to a string
// representation of a url parameter.
func (p params) Encode() string {
	return url.Values(p).Encode()
}

func pathFromContextType(contextType string) string {
	switch contextType {
	case "Course":
		return "courses"
	case "User":
		return "users"
	case "GroupCategory":
		return "group_categories"
	case "Account":
		return "accounts"
	default:
		return ""
	}
}

var _ encoder = (*params)(nil)

func filenameContentType(filename string) string {
	ext := filepath.Ext(filename)
	if ext[0] == '.' {
		ext = ext[1:]
	}
	switch ext {
	case "pdf":
		return "application/pdf"
	case "doc":
		return "application/msword"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "ppt":
		return "application/vnd.ms-powerpoint"
	case "pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case "xls":
		return "application/vnd.ms-excel"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "zip":
		return "application/zip"
	case "gz":
		return "application/gzip"
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	case "png":
		return "image/png"
	case "jpeg", "jpg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "svg":
		return "image/svg+xml"
	case "html", "htm":
		return "text/html"
	case "cpp", "hpp":
		return "text/x-c++src"
	case "txt":
		return "text/plain"
	}
	return ""
}
