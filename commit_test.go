package buildah

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	imageStorage "github.com/containers/image/v5/storage"
	"github.com/containers/storage"
	storageTypes "github.com/containers/storage/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitLinkedLayers(t *testing.T) {
	// This test cannot be parallized as this uses NewBuilder()
	// which eventually and indirectly accesses a global variable
	// defined in `go-selinux`, this must be fixed at `go-selinux`
	// or builder must enable sometime of locking mechanism i.e if
	// routine is creating Builder other's must wait for it.
	// Tracked here: https://github.com/containers/buildah/issues/5967
	ctx := context.TODO()
	now := time.Now()

	graphDriverName := os.Getenv("STORAGE_DRIVER")
	if graphDriverName == "" {
		graphDriverName = "vfs"
	}
	store, err := storage.GetStore(storageTypes.StoreOptions{
		RunRoot:         t.TempDir(),
		GraphRoot:       t.TempDir(),
		GraphDriverName: graphDriverName,
	})
	require.NoError(t, err, "initializing storage")
	t.Cleanup(func() { _, err := store.Shutdown(true); assert.NoError(t, err) })

	imageName := func(i int) string { return fmt.Sprintf("image%d", i) }
	makeFile := func(base string, size int64) string {
		t.Helper()
		fn := filepath.Join(t.TempDir(), base)
		f, err := os.Create(fn)
		require.NoError(t, err)
		defer f.Close()
		if size == 0 {
			size = 512
		}
		_, err = io.CopyN(f, rand.Reader, size)
		require.NoErrorf(t, err, "writing payload file %d", base)
		return f.Name()
	}
	makeArchive := func(base string, size int64) string {
		t.Helper()
		file := makeFile(base, size)
		archiveDir := t.TempDir()
		st, err := os.Stat(file)
		require.NoError(t, err)
		archiveName := filepath.Join(archiveDir, filepath.Base(file))
		f, err := os.Create(archiveName)
		require.NoError(t, err)
		defer f.Close()
		tw := tar.NewWriter(f)
		defer tw.Close()
		hdr, err := tar.FileInfoHeader(st, "")
		require.NoErrorf(t, err, "building tar header for %s", file)
		err = tw.WriteHeader(hdr)
		require.NoErrorf(t, err, "writing tar header for %s", file)
		f, err = os.Open(file)
		require.NoError(t, err)
		defer f.Close()
		_, err = io.Copy(tw, f)
		require.NoErrorf(t, err, "writing tar payload for %s", file)
		return archiveName
	}
	layerNumber := 0

	// Build a from-scratch image with one layer.
	builderOptions := BuilderOptions{
		FromImage: "scratch",
		NamespaceOptions: []NamespaceOption{{
			Name: string(rspec.NetworkNamespace),
			Host: true,
		}},
		SystemContext: &testSystemContext,
	}
	b, err := NewBuilder(ctx, store, builderOptions)
	require.NoError(t, err, "creating builder")
	b.SetCreatedBy(imageName(layerNumber))
	firstFile := makeFile("file0", 0)
	err = b.Add("/", false, AddAndCopyOptions{}, firstFile)
	require.NoError(t, err, "adding", firstFile)
	commitOptions := CommitOptions{
		SystemContext: &testSystemContext,
	}
	ref, err := imageStorage.Transport.ParseStoreReference(store, imageName(layerNumber))
	require.NoError(t, err, "parsing reference for to-be-committed image", imageName(layerNumber))
	_, _, _, err = b.Commit(ctx, ref, commitOptions)
	require.NoError(t, err, "committing", imageName(layerNumber))

	// Build another image based on the first with not much in its layer.
	builderOptions.FromImage = imageName(layerNumber)
	layerNumber++
	b, err = NewBuilder(ctx, store, builderOptions)
	require.NoError(t, err, "creating builder")
	b.SetCreatedBy(imageName(layerNumber))
	secondFile := makeFile("file1", 0)
	err = b.Add("/", false, AddAndCopyOptions{}, secondFile)
	require.NoError(t, err, "adding", secondFile)
	commitOptions = CommitOptions{
		SystemContext: &testSystemContext,
	}
	ref, err = imageStorage.Transport.ParseStoreReference(store, imageName(layerNumber))
	require.NoError(t, err, "parsing reference for to-be-committed image", imageName(layerNumber))
	_, _, _, err = b.Commit(ctx, ref, commitOptions)
	require.NoError(t, err, "committing", imageName(layerNumber))

	// Build a third image with two layers on either side of its read-write layer.
	builderOptions.FromImage = imageName(layerNumber)
	layerNumber++
	b, err = NewBuilder(ctx, store, builderOptions)
	require.NoError(t, err, "creating builder")
	thirdFile := makeFile("file2", 0)
	fourthArchiveFile := makeArchive("file3", 0)
	fifthFile := makeFile("file4", 0)
	sixthFile := makeFile("file5", 0)
	seventhArchiveFile := makeArchive("file6", 0)
	eighthFile := makeFile("file7", 0)
	ninthArchiveFile := makeArchive("file8", 0)
	err = b.Add("/", false, AddAndCopyOptions{}, sixthFile)
	require.NoError(t, err, "adding", sixthFile)
	b.SetCreatedBy(imageName(layerNumber + 3))
	b.AddPrependedLinkedLayer(nil, imageName(layerNumber), "", "", filepath.Dir(thirdFile))
	commitOptions = CommitOptions{
		PrependedLinkedLayers: []LinkedLayer{
			{
				BlobPath: fourthArchiveFile,
				History: v1.History{
					Created:   &now,
					CreatedBy: imageName(layerNumber + 1),
				},
			},
			{
				BlobPath: filepath.Dir(fifthFile),
				History: v1.History{
					Created:   &now,
					CreatedBy: imageName(layerNumber + 2),
				},
			},
		},
		AppendedLinkedLayers: []LinkedLayer{
			{
				BlobPath: seventhArchiveFile,
				History: v1.History{
					Created:   &now,
					CreatedBy: imageName(layerNumber + 4),
				},
			},
			{
				BlobPath: filepath.Dir(eighthFile),
				History: v1.History{
					Created:   &now,
					CreatedBy: imageName(layerNumber + 5),
				},
			},
		},
		SystemContext: &testSystemContext,
	}
	b.AddAppendedLinkedLayer(nil, imageName(layerNumber+6), "", "", ninthArchiveFile)
	ref, err = imageStorage.Transport.ParseStoreReference(store, imageName(layerNumber))
	require.NoError(t, err, "parsing reference for to-be-committed image", imageName(layerNumber))
	_, _, _, err = b.Commit(ctx, ref, commitOptions)
	require.NoError(t, err, "committing", imageName(layerNumber))

	// Build one last image based on the previous one.
	builderOptions.FromImage = imageName(layerNumber)
	layerNumber += 7
	b, err = NewBuilder(ctx, store, builderOptions)
	require.NoError(t, err, "creating builder")
	b.SetCreatedBy(imageName(layerNumber))
	tenthFile := makeFile("file9", 0)
	err = b.Add("/", false, AddAndCopyOptions{}, tenthFile)
	require.NoError(t, err, "adding", tenthFile)
	commitOptions = CommitOptions{
		SystemContext: &testSystemContext,
	}
	ref, err = imageStorage.Transport.ParseStoreReference(store, imageName(layerNumber))
	require.NoError(t, err, "parsing reference for to-be-committed image", imageName(layerNumber))
	_, _, _, err = b.Commit(ctx, ref, commitOptions)
	require.NoError(t, err, "committing", imageName(layerNumber))

	// Get set to examine this image.  At this point, each history entry
	// should just have "image%d" as its CreatedBy field, and each layer
	// should have the corresponding file (and nothing else) in it.
	src, err := ref.NewImageSource(ctx, &testSystemContext)
	require.NoError(t, err, "opening image source")
	defer src.Close()
	img, err := ref.NewImage(ctx, &testSystemContext)
	require.NoError(t, err, "opening image")
	defer img.Close()
	config, err := img.OCIConfig(ctx)
	require.NoError(t, err, "reading config in OCI format")
	require.Len(t, config.History, 10, "history length")
	for i := range config.History {
		require.Equal(t, fmt.Sprintf("image%d", i), config.History[i].CreatedBy, "history createdBy is off")
	}
	require.Len(t, config.RootFS.DiffIDs, 10, "diffID list")

	layerContents := func(archive io.ReadCloser) []string {
		var contents []string
		defer archive.Close()
		tr := tar.NewReader(archive)
		entry, err := tr.Next()
		for entry != nil {
			contents = append(contents, entry.Name)
			if err != nil {
				break
			}
			entry, err = tr.Next()
		}
		require.ErrorIs(t, err, io.EOF)
		return contents
	}
	infos, err := img.LayerInfosForCopy(ctx)
	require.NoError(t, err, "getting layer infos")
	require.Len(t, infos, 10)
	for i, blobInfo := range infos {
		func() {
			t.Helper()
			rc, _, err := src.GetBlob(ctx, blobInfo, nil)
			require.NoError(t, err, "getting blob", i)
			defer rc.Close()
			contents := layerContents(rc)
			require.Len(t, contents, 1)
			require.Equal(t, fmt.Sprintf("file%d", i), contents[0])
		}()
	}
}
