package paste

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/aexvir/skladka/internal/errors"
)

type Paste struct {
	Reference  string     `json:"reference"`
	Owner      *string    `json:"owner"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Mimetype   *string    `json:"mimetype"`
	Syntax     string     `json:"syntax"`
	Tags       []string   `json:"tags"`
	Creation   time.Time  `json:"creation"`
	Expiration *time.Time `json:"expiration"`
	Public     bool       `json:"public"`
	Password   *string    `json:"password"`
	Views      int        `json:"views"`
}

func (p Paste) IsBase64Encoded() bool {
	return p.Syntax == "base64encode"
}

func (p Paste) FileName() string {
	ext := p.FileExtension()
	return strings.TrimSuffix(p.Title, ext) + ext
}

func (p Paste) FileExtension() string {
	ext := filepath.Ext(p.Title)
	if ext != "" {
		return ext
	}

	mime := MimeUknown
	if p.Mimetype != nil {
		mime = *p.Mimetype
	}

	if ext, ok := MimetypeFileExtension[mime]; ok {
		return ext
	}

	return ".txt"
}

func (p Paste) SizeBytes() float64 {
	return float64(len([]byte(p.Content)))
}

// Validate checks if the paste meets all validation rules.
// It returns an error if any rule is violated.
func (p Paste) Validate() error {
	var errs []error

	// content is required and must not be empty
	if strings.TrimSpace(p.Content) == "" {
		errs = append(errs, errors.New("can't create a paste without content"))
	}

	// title if provided must not be empty
	if p.Title != "" && strings.TrimSpace(p.Title) == "" {
		errs = append(errs, errors.New("title if provided must not be empty"))
	}

	// tags if provided must not be empty
	for _, tag := range p.Tags {
		if strings.TrimSpace(tag) == "" {
			errs = append(errs, errors.New("empty tags are not allowed"))
		}
	}

	// expiration if provided must be in the future
	if p.Expiration != nil && p.Expiration.Before(time.Now()) {
		errs = append(errs, errors.New("expiration must be in the future"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
