package paste

import (
	"strings"
	"time"

	"github.com/aexvir/skladka/internal/errors"
)

type Paste struct {
	Reference  string     `json:"reference"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Syntax     string     `json:"syntax"`
	Tags       []string   `json:"tags"`
	Creation   time.Time  `json:"creation"`
	Expiration *time.Time `json:"expiration"`
	Public     bool       `json:"public"`
	Password   *string    `json:"password"`
	Views      int        `json:"views"`
}

// Validate checks if the paste meets all validation rules.
// It returns an error if any rule is violated.
func (p *Paste) Validate() error {
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
