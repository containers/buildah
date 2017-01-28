package policyconfiguration

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/containers/image/docker/reference"
)

// DockerReferenceIdentity returns a string representation of the reference, suitable for policy lookup,
// as a backend for ImageReference.PolicyConfigurationIdentity.
// The reference must satisfy !reference.IsNameOnly().
func DockerReferenceIdentity(ref reference.Named) (string, error) {
	res := ref.FullName()
	tagged, isTagged := ref.(reference.NamedTagged)
	digested, isDigested := ref.(reference.Canonical)
	switch {
	case isTagged && isDigested: // This should not happen, docker/reference.ParseNamed drops the tag.
		return "", errors.Errorf("Unexpected Docker reference %s with both a name and a digest", ref.String())
	case !isTagged && !isDigested: // This should not happen, the caller is expected to ensure !reference.IsNameOnly()
		return "", errors.Errorf("Internal inconsistency: Docker reference %s with neither a tag nor a digest", ref.String())
	case isTagged:
		res = res + ":" + tagged.Tag()
	case isDigested:
		res = res + "@" + digested.Digest().String()
	default: // Coverage: The above was supposed to be exhaustive.
		return "", errors.New("Internal inconsistency, unexpected default branch")
	}
	return res, nil
}

// DockerReferenceNamespaces returns a list of other policy configuration namespaces to search,
// as a backend for ImageReference.PolicyConfigurationIdentity.
// The reference must satisfy !reference.IsNameOnly().
func DockerReferenceNamespaces(ref reference.Named) []string {
	// Look for a match of the repository, and then of the possible parent
	// namespaces. Note that this only happens on the expanded host names
	// and repository names, i.e. "busybox" is looked up as "docker.io/library/busybox",
	// then in its parent "docker.io/library"; in none of "busybox",
	// un-namespaced "library" nor in "" supposedly implicitly representing "library/".
	//
	// ref.FullName() == ref.Hostname() + "/" + ref.RemoteName(), so the last
	// iteration matches the host name (for any namespace).
	res := []string{}
	name := ref.FullName()
	for {
		res = append(res, name)

		lastSlash := strings.LastIndex(name, "/")
		if lastSlash == -1 {
			break
		}
		name = name[:lastSlash]
	}
	return res
}
