package buildah

import (
	"os/user"
	"strconv"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// TODO: we should doing these lookups using data that's actually in the container.
func getUser(username string) (specs.User, error) {
	if username == "" {
		return specs.User{}, nil
	}
	runuser, err := user.Lookup(username)
	if err != nil {
		return specs.User{}, err
	}
	uid, err := strconv.ParseUint(runuser.Uid, 10, 32)
	if err != nil {
		return specs.User{}, nil
	}
	gid, err := strconv.ParseUint(runuser.Gid, 10, 32)
	if err != nil {
		return specs.User{}, nil
	}
	u := specs.User{
		UID:      uint32(uid),
		GID:      uint32(gid),
		Username: username,
	}
	return u, nil
}
