package dockerclient

import (
	"archive/tar"
	"context"
	"io"
	"io/ioutil"

	docker "github.com/fsouza/go-dockerclient"
	"k8s.io/klog"
)

type DirectoryCheck interface {
	IsDirectory(path string) (bool, error)
}

type directoryCheck struct {
	containerID string
	client      *docker.Client
}

func newDirectoryCheck(client *docker.Client, containerID string) *directoryCheck {
	return &directoryCheck{
		containerID: containerID,
		client:      client,
	}
}

func (c *directoryCheck) IsDirectory(path string) (bool, error) {
	if path == "/" || path == "." || path == "./" {
		return true, nil
	}

	dir, err := isContainerPathDirectory(c.client, c.containerID, path)
	if err != nil {
		return false, err
	}

	return dir, nil
}

func isContainerPathDirectory(client *docker.Client, containerID, path string) (bool, error) {
	pr, pw := io.Pipe()
	defer pw.Close()
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		err := client.DownloadFromContainer(containerID, docker.DownloadFromContainerOptions{
			OutputStream: pw,
			Path:         path,
			Context:      ctx,
		})
		if err != nil {
			if apiErr, ok := err.(*docker.Error); ok && apiErr.Status == 404 {
				klog.V(4).Infof("path %s did not exist in container %s: %v", path, containerID, err)
				err = nil
			}
			if err != nil && err != context.Canceled {
				klog.V(6).Infof("error while checking directory contents for container %s at path %s: %v", containerID, path, err)
			}
		}
		pw.CloseWithError(err)
	}()

	tr := tar.NewReader(pr)

	h, err := tr.Next()
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		cancel()
		return false, err
	}

	klog.V(4).Infof("Retrieved first header from container %s at path %s: %#v", containerID, path, h)

	// take the remainder of the input and discard it
	go func() {
		cancel()
		n, err := io.Copy(ioutil.Discard, pr)
		if n > 0 || err != nil {
			klog.V(6).Infof("Discarded %d bytes from end of container directory check, and got error: %v", n, err)
		}
	}()

	return h.FileInfo().IsDir(), nil
}
