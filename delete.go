package buildah

import (
	"fmt"
	"os"
)

func (b *Builder) Delete() error {
	for _, link := range b.Links {
		if err := os.Remove(link); err != nil {
			return fmt.Errorf("error removing symlink %q: %v", link, err)
		}
	}
	b.Links = nil

	if err := b.store.DeleteContainer(b.ContainerID); err != nil {
		return fmt.Errorf("error deleting build container: %v", err)
	}
	b.MountPoint = ""
	b.Container = ""
	b.ContainerID = ""
	return nil
}
