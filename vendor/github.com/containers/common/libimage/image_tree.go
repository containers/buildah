package libimage

import (
	"fmt"
	"strings"

	"github.com/docker/go-units"
)

const (
	imageTreeMiddleItem   = "├── "
	imageTreeContinueItem = "│   "
	imageTreeLastItem     = "└── "
)

// Tree generates a tree for the specified image and its layers.  Use
// `traverseChildren` to traverse the layers of all children.  By default, only
// layers of the image are printed.
func (i *Image) Tree(traverseChildren bool) (*strings.Builder, error) {
	// NOTE: a string builder prevents us from copying to much data around
	// and compile the string when and where needed.
	sb := &strings.Builder{}

	// First print the pretty header for the target image.
	size, err := i.Size()
	if err != nil {
		return nil, err
	}
	repoTags, err := i.RepoTags()
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(sb, "Image ID: %s\n", i.ID()[:12])
	fmt.Fprintf(sb, "Tags:     %s\n", repoTags)
	fmt.Fprintf(sb, "Size:     %v\n", units.HumanSizeWithPrecision(float64(size), 4))
	if i.TopLayer() != "" {
		fmt.Fprintf(sb, "Image Layers\n")
	} else {
		fmt.Fprintf(sb, "No Image Layers\n")
	}

	layerTree, err := i.runtime.layerTree()
	if err != nil {
		return nil, err
	}

	imageNode := layerTree.node(i.TopLayer())
	// Traverse the entire tree down to all children.
	if traverseChildren {
		return imageTreeTraverseChildren(sb, imageNode, "", true)
	}

	// Walk all layers of the image and assemlbe their data.
	for parentNode := imageNode.parent; parentNode != nil; parentNode = parentNode.parent {
		indent := imageTreeMiddleItem
		if parentNode.parent == nil {
			indent = imageTreeLastItem
		}

		var tags string
		repoTags, err := parentNode.repoTags()
		if err != nil {
			return nil, err
		}
		if len(repoTags) > 0 {
			tags = fmt.Sprintf(" Top Layer of: %s", repoTags)
		}
		fmt.Fprintf(sb, "%s ID: %s Size: %7v%s\n", indent, parentNode.layer.ID[:12], units.HumanSizeWithPrecision(float64(parentNode.layer.UncompressedSize), 4), tags)
	}

	return sb, nil
}

func imageTreeTraverseChildren(sb *strings.Builder, node *layerNode, prefix string, last bool) (*strings.Builder, error) {
	numChildren := len(node.children)
	if numChildren == 0 {
		return sb, nil
	}
	sb.WriteString(prefix)

	intend := imageTreeMiddleItem
	if !last {
		prefix += imageTreeContinueItem
	} else {
		intend = imageTreeLastItem
		prefix += " "
	}

	for i := range node.children {
		child := node.children[i]
		var tags string
		repoTags, err := child.repoTags()
		if err != nil {
			return nil, err
		}
		if len(repoTags) > 0 {
			tags = fmt.Sprintf(" Top Layer of: %s", repoTags)
		}
		fmt.Fprintf(sb, "%sID: %s Size: %7v%s\n", intend, child.layer.ID[:12], units.HumanSizeWithPrecision(float64(child.layer.UncompressedSize), 4), tags)
		sb, err = imageTreeTraverseChildren(sb, child, prefix, i == numChildren-1)
		if err != nil {
			return nil, err
		}
	}

	return sb, nil
}
