package buildah

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/containers/storage"
	"github.com/containers/storage/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	debug := false
	if InitReexec() {
		return
	}
	flag.BoolVar(&debug, "debug", false, "turn on debug logging")
	flag.Parse()
	logrus.SetLevel(logrus.ErrorLevel)
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	os.Exit(m.Run())
}

func TestOpenBuilderCommonBuildOpts(t *testing.T) {
	ctx := context.TODO()
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:         t.TempDir(),
		GraphRoot:       t.TempDir(),
		GraphDriverName: "vfs",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _, err := store.Shutdown(true); assert.NoError(t, err) })
	b, err := NewBuilder(ctx, store, BuilderOptions{})
	require.NoError(t, err)
	require.NotNil(t, b.CommonBuildOpts)
	b.CommonBuildOpts = nil
	builderContainerID := b.ContainerID
	err = b.Save()
	require.NoError(t, err)
	b, err = OpenBuilder(store, builderContainerID)
	require.NoError(t, err)
	require.NotNil(t, b.CommonBuildOpts)
	builders, err := OpenAllBuilders(store)
	require.NoError(t, err)
	for _, b := range builders {
		require.NotNil(t, b.CommonBuildOpts)
	}
	imageID, _, _, err := b.Commit(ctx, nil, CommitOptions{})
	require.NoError(t, err)
	b, err = ImportBuilderFromImage(ctx, store, ImportFromImageOptions{
		Image: imageID,
	})
	require.NoError(t, err)
	require.NotNil(t, b.CommonBuildOpts)
	container, err := store.CreateContainer("", nil, imageID, "", "", &storage.ContainerOptions{})
	require.NoError(t, err)
	require.NotNil(t, container)
	b, err = ImportBuilder(ctx, store, ImportOptions{
		Container: container.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, b.CommonBuildOpts)
}
