package buildah

import (
	"os"

	"github.com/Sirupsen/logrus"
)

// Unmount unmounts a build container, removing any links which have been made
// to its root.
func (b *Builder) Unmount() error {
	err := b.store.Unmount(b.ContainerID)
	if err == nil {
		for _, l := range b.Links {
			if err := os.Remove(l); err != nil {
				logrus.Errorf("error removing symbolic link %q: %v", l, err)
			}
		}
		b.Links = []string{}
		err = b.Save()
	}
	return err
}
