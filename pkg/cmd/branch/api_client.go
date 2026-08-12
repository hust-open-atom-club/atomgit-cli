package branch

import (
	"net/url"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func parseRepository(repository string) (string, string, error) {
	parsed, err := cmdutil.ParseRepository(repository)
	if err != nil {
		return "", "", err
	}
	return parsed.Owner, parsed.Name, nil
}

func escapePathSegment(value string) string {
	return url.PathEscape(value)
}
