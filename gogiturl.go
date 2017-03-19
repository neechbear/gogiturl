package gogiturl

import (
	"errors"
	"net/url"
	"strings"
)

// GitURL prepresents Git remote repository URL, parsed by net/url, with
// extended support for Git and SCP-like remote syntax of user@host:path.
type GitURL struct {
	url.URL
}

// Parse parses rawurl into a URL structure.
func Parse(rawurl string) (*GitURL, error) {
	u, err := parse(rawurl, rawurl)
	return u, err
}

// Wrapper around net/url.Parse to try and support Git@/scp style remotes.
func parse(rawurl string, origrawurl string) (*GitURL, error) {
	url, err := url.Parse(rawurl)

	if err == nil {
		// If we needed to munge the raw URL, then strip the first character
		// from the Path property (a colon : delimier turned in to a slash /).
		if rawurl != origrawurl {
			url.Path = url.Path[1:]
		}

		return &GitURL{*url}, err
	}

	// From this point on we are going to try and munge the rawurl that we were
	// passed, and see if we can parse them as any of the following Git@/scp
	// type "URLs", all of which are assumed to use a URL scheme of ssh://, and
	// contain at least host and path parts, delimited by a colon. They may
	// optionally include a username part delimited with an @ symbol (but no
	// password).

	// host.xz:/path/to/repo.git/
	// host.xz:~user/path/to/repo.git/
	// host.xz:path/to/repo.git
	// [d:e:a:d::1]:/path/to/repo.git/
	// 10.10.10.10:/path/to/repo.git/
	// user@10.10.10.10:/path/to/repo.git/
	// user@[d:e:a:d::1]:/path/to/repo.git/
	// user@host.xz:/path/to/repo.git/
	// user@host.xz:~user/path/to/repo.git/
	// user@host.xz:path/to/repo.git

	// If we failed to parse the URL, but it has a URL scheme or doesn't
	// contain a colon, then it doesn't match our edge case git@/scp format, so
	// we will give up.
	if strings.Index(rawurl, ":") < 0 {
		return nil, err
	}
	scheme, _, schemeerr := getScheme(rawurl)
	if schemeerr == nil || scheme != "" {
		return nil, err
	}

	// Hopefully an edge case git@/scp type "URL" syntax.
	i := strings.Index(rawurl, "]") // Does it look like we have an IPv6 host
	j := 0
	if i < 0 {
		j = strings.Index(rawurl, ":")
	} else { // Probably IPv6 hostname.
		j = strings.Index(rawurl[i:], ":")
	}

	if j < 0 { // No colon in URL, so probably bogus.
		err = errors.New("No colon (:) in URL to delimit host:path boundary; " +
			"unable to munge Git remote edge case")
		return nil, err
	}

	if i >= 0 { // Probably IPv6 hostname.
		j = j + i // Add IPv6 hostname index offset to colon (:) offset
	}

	// Munge a URL with forced ssh:// scheme, replacing the delimiting
	// hostname:path colon (:) with a slash (/).
	mungedurl := "ssh://" +
		rawurl[:j] +
		strings.Replace(rawurl[j:], ":", "/", 1)

	// Reparse our munged URL.
	return parse(mungedurl, origrawurl)
}

// Taken almost verbatim from net/url.
func getScheme(rawurl string) (scheme, path string, err error) {
	for i := 0; i < len(rawurl); i++ {
		c := rawurl[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return "", rawurl, errors.New("missing protocol scheme")
			}
		case c == ':':
			if i == 0 {
				return "", "", errors.New("missing protocol scheme")
			}
			return rawurl[:i], rawurl[i+1:], nil
		default:
			// we have encountered an invalid character,
			// so there is no valid scheme
			return "", rawurl, errors.New("missing protocol scheme")
		}
	}
	return "", rawurl, errors.New("missing protocol scheme")
}
