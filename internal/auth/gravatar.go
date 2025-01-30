package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func GravatarUrlFromEmail(email string) string {
	checksum := sha256.Sum256(
		[]byte(strings.ToLower(strings.TrimSpace(email))),
	)
	hash := hex.EncodeToString(checksum[:])

	return GravatarUrlFromHash(hash)
}

func GravatarUrlFromUsername(username string) string {
	placeholder := "https://gravatar.com/avatar/00000000000000000000000000000000?s=512"

	resp, err := http.Get(fmt.Sprintf("https://gravatar.com/%s", username))
	if err != nil {
		return placeholder
	}
	defer resp.Body.Close()

	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return placeholder
	}

	avatar := regexp.MustCompile("https://(?:[a-zA-Z0-9.]+)?gravatar\\.com/avatar/([a-zA-Z0-9]{64})")
	matches := avatar.FindSubmatch(html)
	if len(matches) < 2 {
		return placeholder
	}

	return GravatarUrlFromHash(string(matches[1]))
}

func GravatarUrlFromHash(hash string) string {
	addr := url.URL{
		Scheme: "https",
		Host:   "gravatar.com",
		Path:   "/avatar/" + hash,
	}

	query := make(url.Values)
	query.Add("s", "512")
	// todo: check if anything else

	addr.RawQuery = query.Encode()

	return addr.String()
}
