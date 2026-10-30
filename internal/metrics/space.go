package metrics

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// Files and folders smaller than this are summarised as one "other" row so a
// whole-volume tree stays small enough to keep in memory.
const spaceDetailBytes = 1_000_000

// SpaceProgress is updated while a scan runs; Path is the folder being read.
type SpaceProgress struct {
	Files   atomic.Int64
	Bytes   atomic.Int64
	Skipped atomic.Int64
	Path    atomic.Pointer[string]
}

// SpaceEntry is one row of the space browser. Other rows summarise Files small
// items and cannot be opened or removed.
type SpaceEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	Files     int64  `json:"files"`
	Directory bool   `json:"directory"`
	Other     bool   `json:"other"`
	Partial   bool   `json:"partial"`
	Trashable bool   `json:"trashable"`
}

type SpaceListing struct {
	Entry     SpaceEntry   `json:"entry"`
	Ancestors []SpaceEntry `json:"ancestors"`
	Children  []SpaceEntry `json:"children"`
}

type spaceNode struct {
	name       string
	parent     *spaceNode
	children   []*spaceNode
	bytes      int64
	files      int64
	otherBytes int64
	otherCount int64
	dir        bool
	partial    bool
	id         FileID
}

// SpaceTree holds allocated sizes for one volume. It is not safe for concurrent use.
type SpaceTree struct {
	root *spaceNode
	path string
}

// ScanSpaceTree measures allocated bytes below root in parallel. It does not
// follow symbolic links, leave the volumes of root and the home directory, or
// count hard-linked files twice. Unreadable entries mark their folder partial.
func ScanSpaceTree(ctx context.Context, root string, skip []string, progress *SpaceProgress) (*SpaceTree, error) {
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	devices := map[uint64]bool{}
	if id, ok := FileIdentity(info); ok {
		devices[id.Device] = true
	}
	if home, err := os.UserHomeDir(); err == nil {
		if info, err := os.Lstat(home); err == nil {
			if id, ok := FileIdentity(info); ok {
				devices[id.Device] = true
			}
		}
	}
	skipped := map[string]bool{}
	for _, path := range skip {
		skipped[filepath.Clean(path)] = true
	}
	w := &spaceWalker{ctx: ctx, progress: progress, devices: devices, skip: skipped, links: map[FileID]bool{}, slots: make(chan struct{}, 4*runtime.NumCPU())}
	top := &spaceNode{name: root, dir: true}
	top.id, _ = FileIdentity(info)
	w.wait.Add(1)
	w.visit(top, root)
	w.wait.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	finish(top)
	return &SpaceTree{root: top, path: root}, nil
}

type spaceWalker struct {
	ctx      context.Context
	progress *SpaceProgress
	devices  map[uint64]bool
	skip     map[string]bool
	linkMu   sync.Mutex
	links    map[FileID]bool
	slots    chan struct{}
	wait     sync.WaitGroup
}

func (w *spaceWalker) visit(node *spaceNode, path string) {
	defer w.wait.Done()
	if w.ctx.Err() != nil {
		return
	}
	w.progress.Path.Store(&path)
	dir, err := os.Open(path)
	if err != nil {
		node.partial = true
		w.progress.Skipped.Add(1)
		return
	}
	entries, err := dir.ReadDir(-1)
	dir.Close()
	if err != nil {
		node.partial = true
		w.progress.Skipped.Add(1)
	}
	var folders []*spaceNode
	for _, entry := range entries {
		if w.ctx.Err() != nil {
			return
		}
		kind := entry.Type()
		if kind&fs.ModeSymlink != 0 || !kind.IsDir() && !kind.IsRegular() {
			continue
		}
		child := filepath.Join(path, entry.Name())
		info, err := os.Lstat(child)
		if err != nil {
			node.partial = true
			w.progress.Skipped.Add(1)
			continue
		}
		id, identified := FileIdentity(info)
		if info.IsDir() {
			if w.skip[child] || identified && len(w.devices) > 0 && !w.devices[id.Device] {
				continue
			}
			folder := &spaceNode{name: entry.Name(), parent: node, dir: true, id: id}
			node.children = append(node.children, folder)
			folders = append(folders, folder)
			continue
		}
		if !info.Mode().IsRegular() || identified && linkCount(info) > 1 && w.seen(id) {
			continue
		}
		size := allocatedBytes(info)
		w.progress.Files.Add(1)
		w.progress.Bytes.Add(size)
		node.files++
		if size >= spaceDetailBytes {
			node.children = append(node.children, &spaceNode{name: entry.Name(), parent: node, bytes: size, files: 1, id: id})
		} else {
			node.otherBytes += size
			node.otherCount++
		}
	}
	for _, folder := range folders {
		w.wait.Add(1)
		path := filepath.Join(path, folder.name)
		select {
		case w.slots <- struct{}{}:
			go func() { w.visit(folder, path); <-w.slots }()
		default:
			w.visit(folder, path)
		}
	}
}

func (w *spaceWalker) seen(id FileID) bool {
	w.linkMu.Lock()
	defer w.linkMu.Unlock()
	if w.links[id] {
		return true
	}
	w.links[id] = true
	return false
}

// finish totals each folder, folds small folders into "other" and sorts by size.
func finish(node *spaceNode) {
	kept := node.children[:0]
	node.bytes = node.otherBytes
	for _, child := range node.children {
		if child.dir {
			finish(child)
			node.files += child.files
			node.partial = node.partial || child.partial
		}
		node.bytes += child.bytes
		if child.dir && child.bytes < spaceDetailBytes && !child.partial {
			node.otherBytes += child.bytes
			node.otherCount++
			continue
		}
		kept = append(kept, child)
	}
	node.children = slices.Clip(kept)
	slices.SortStableFunc(node.children, bySize)
}

func bySize(a, b *spaceNode) int {
	switch {
	case a.bytes > b.bytes:
		return -1
	case a.bytes < b.bytes:
		return 1
	}
	return strings.Compare(a.name, b.name)
}

func (t *SpaceTree) Root() string { return t.path }
func (t *SpaceTree) Bytes() int64 { return t.root.bytes }
func (t *SpaceTree) Files() int64 { return t.root.files }

func (t *SpaceTree) node(path string) *spaceNode {
	rel, err := filepath.Rel(t.path, filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	node := t.root
	if rel == "." {
		return node
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		next := (*spaceNode)(nil)
		for _, child := range node.children {
			if child.name == part {
				next = child
				break
			}
		}
		if next == nil {
			return nil
		}
		node = next
	}
	return node
}

func (t *SpaceTree) pathOf(node *spaceNode) string {
	var parts []string
	for ; node.parent != nil; node = node.parent {
		parts = append(parts, node.name)
	}
	slices.Reverse(parts)
	return filepath.Join(append([]string{t.path}, parts...)...)
}

func (t *SpaceTree) entry(node *spaceNode) SpaceEntry {
	return SpaceEntry{Name: node.name, Path: t.pathOf(node), Bytes: node.bytes, Files: node.files, Directory: node.dir, Partial: node.partial}
}

// Find returns the measured entry at path.
func (t *SpaceTree) Find(path string) (SpaceEntry, bool) {
	node := t.node(path)
	if node == nil {
		return SpaceEntry{}, false
	}
	return t.entry(node), true
}

// Identity returns the file identity recorded during the scan.
func (t *SpaceTree) Identity(path string) (FileID, bool) {
	node := t.node(path)
	if node == nil {
		return FileID{}, false
	}
	return node.id, true
}

// Listing returns a folder, its ancestors and up to limit children by size;
// remaining and small items are summarised in a final "other" row.
func (t *SpaceTree) Listing(path string, limit int) (SpaceListing, error) {
	node := t.node(path)
	if node == nil || !node.dir {
		return SpaceListing{}, fs.ErrNotExist
	}
	listing := SpaceListing{Entry: t.entry(node), Children: []SpaceEntry{}}
	for parent := node.parent; parent != nil; parent = parent.parent {
		listing.Ancestors = append([]SpaceEntry{t.entry(parent)}, listing.Ancestors...)
	}
	other := SpaceEntry{Other: true, Bytes: node.otherBytes, Files: node.otherCount}
	for i, child := range node.children {
		if i < limit {
			listing.Children = append(listing.Children, t.entry(child))
			continue
		}
		other.Bytes += child.bytes
		other.Files++
	}
	if other.Files > 0 {
		listing.Children = append(listing.Children, other)
	}
	return listing, nil
}

// Largest returns up to limit individually measured files, largest first,
// excluding the subtree at exclude.
func (t *SpaceTree) Largest(limit int, exclude string) []SpaceEntry {
	skip := t.node(exclude)
	var files []*spaceNode
	var walk func(*spaceNode)
	walk = func(node *spaceNode) {
		if node == skip && exclude != "" {
			return
		}
		for _, child := range node.children {
			if child.dir {
				walk(child)
			} else {
				files = append(files, child)
			}
		}
	}
	walk(t.root)
	slices.SortFunc(files, bySize)
	entries := []SpaceEntry{}
	for _, file := range files[:min(limit, len(files))] {
		entries = append(entries, t.entry(file))
	}
	return entries
}

func (n *spaceNode) adjust(bytes, files int64) {
	for node := n; node != nil; node = node.parent {
		node.bytes += bytes
		node.files += files
	}
}

// Move re-parents the entry at path under folder, as moving to the trash does.
func (t *SpaceTree) Move(path, folder string) bool {
	node, target := t.node(path), t.node(folder)
	if node == nil || node.parent == nil {
		return false
	}
	node.parent.children = slices.DeleteFunc(node.parent.children, func(child *spaceNode) bool { return child == node })
	node.parent.adjust(-node.bytes, -node.files)
	node.parent = nil
	if target == nil || !target.dir {
		return true
	}
	node.parent = target
	target.children = append(target.children, node)
	target.adjust(node.bytes, node.files)
	slices.SortStableFunc(target.children, bySize)
	return true
}

// Clear empties the folder at path, as emptying the trash does.
func (t *SpaceTree) Clear(path string) {
	node := t.node(path)
	if node == nil {
		return
	}
	node.adjust(-node.bytes, -node.files)
	node.children, node.otherBytes, node.otherCount, node.partial = nil, 0, 0, false
}
