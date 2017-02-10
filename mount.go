package buildah

func (b *Builder) Mount(label string) (string, error) {
	mountpoint, err := b.store.Mount(b.ContainerID, label)
	if err != nil {
		return "", err
	}
	b.MountPoint = mountpoint

	present := false
	for _, m := range b.Mounts {
		if m == mountpoint {
			present = true
			break
		}
	}
	if !present {
		b.Mounts = append(b.Mounts, mountpoint)
	}

	err = b.Save()
	if err != nil {
		return "", err
	}
	return mountpoint, nil
}
