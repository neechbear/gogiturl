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
	munged := false
	scheme, urlremainder, err := getscheme(rawurl)

	// If there's a scheme:// but nothing left to the URL other than the scheme,
	// (the urlremainder), then this is junk.
	if err == nil && urlremainder == "" {
		return nil, errors.New("malformed URL contains scheme only")
	}

	// If there's no scheme:// in the URL, then we may need to munge it.
	if err != nil || scheme == "" {
		if strings.Index(rawurl, ":") < 0 {
			// TODO: Merge this particular munging into mungeGitURL().
			rawurl = "file:///" + rawurl
		} else {
			mungedurl, err := mungeGitURL(rawurl)
			if err != nil {
				return nil, err
			}
			rawurl = mungedurl
		}
		munged = true
	}

	url, err := url.Parse(rawurl)
	if err == nil {
		// If we munged rawurl, then revert back the first character of the Path
		// property (a colon : delimier turned in to a slash /).
		if munged {
			url.Path = url.Path[1:]
		}
	}

	return &GitURL{*url}, err
}

// Munge the given Git@/scp rawurl in to a classical net/url parsable format.
func mungeGitURL(rawurl string) (string, error) {
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

	// TODO: a better IPv6 check would be to look for [ at the start of the
	//       string, or @[ (in the case of a username@), and only then for the
	//       closing ] after that index location.
	i := strings.Index(rawurl, "]") // Does it look like we have an IPv6 host
	j := 0
	if i < 0 {
		j = strings.Index(rawurl, ":")
	} else { // Probably IPv6 hostname.
		j = strings.Index(rawurl[i:], ":")
	}

	if j < 0 { // No colon in URL, so probably bogus.
		return "", errors.New("no colon (:) in URL to delimit host:path " +
			"boundary; unable to munge Git remote edge case")
	}

	if i >= 0 { // Probably IPv6 hostname.
		j = j + i // Add IPv6 hostname index offset to colon (:) offset
	}

	// Munge a URL with forced ssh:// scheme, replacing the delimiting
	// hostname:path colon (:) with a slash (/).
	mungedurl := "ssh://" +
		rawurl[:j] +
		strings.Replace(rawurl[j:], ":", "/", 1)

	return mungedurl, nil
}

// Returns scheme:// and remainder of URL from the given rawurl string.
func getscheme(rawurl string) (string, string, error) {
	urllen := len(rawurl)
	for i := 0; i < urllen; i++ {
		c := rawurl[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return "", "", errors.New("no scheme")
			}
		case c == ':': // end of scheme
			if i > 0 && urllen >= i+3 && rawurl[i+1:i+3] == "//" {
				if urllen >= i+4 {
					return rawurl[:i], rawurl[i+4:], nil
				}
				return rawurl[:i], "", nil
			}
			return "", "", errors.New("no scheme")
		default: // illegal character
			return "", "", errors.New("invalid character in scheme")
		}
	}
	return "", "", errors.New("no scheme")
}
