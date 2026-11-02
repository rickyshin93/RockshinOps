// Package fileindex keeps the names of the files under a folder in memory
// and searches them as the user types: names, not contents, ranked by how
// well they match, how recently they changed and how deep they are.
package fileindex

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

// Rules say what is not indexed: folders with these names anywhere, folders
// at these paths relative to the root, except the places Include matches
// (globs relative to the root), which are indexed although they lie in a
// skipped folder, as received chat files and iCloud Drive lie in Library.
type Rules struct {
	SkipNames []string `json:"skipNames"`
	SkipPaths []string `json:"skipPaths"`
	Include   []string `json:"include"`
}

type Match struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Dir      bool      `json:"dir"`
	Modified time.Time `json:"modified"`
	Score    int       `json:"-"` // on the scale of Score, with recency and depth applied
}

const (
	flagDir uint8 = 1 << iota
	flagDeleted
	flagHidden // a folder leading to an included place: kept for paths, never a result
	flagFolded // the lower-case name differs and is kept in Index.lower
	flagSpelled // the name has Chinese characters, spelled out in Index.spelled
)

// entry is one file or folder; names live in shared byte arenas so a
// million entries take tens of megabytes.
type entry struct {
	parent   int32
	name     uint32
	lower    uint32
	nameLen  uint16
	lowerLen uint16
	mod      uint32 // Unix seconds
	flags    uint8
}

type Index struct {
	root      string
	skipNames map[string]bool
	skipPaths []string
	include   []string

	mu       sync.RWMutex
	entries  []entry
	names    []byte
	lower    []byte
	children map[int32][]int32
	spelled  map[int32][]syllable
	deleted  int
	included []string // root-relative included places found by Build
}

func New(root string, rules Rules) *Index {
	ix := &Index{root: filepath.Clean(root), skipNames: map[string]bool{}, skipPaths: rules.SkipPaths, include: rules.Include}
	for _, n := range rules.SkipNames {
		ix.skipNames[n] = true
	}
	return ix
}

// Len is the number of files and folders indexed.
func (ix *Index) Len() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.entries) - ix.deleted
}

// Roots are the folders to watch: the root, and the included places, which
// a watch of the root does not enter.
func (ix *Index) Roots() []string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	roots := []string{ix.root}
	for _, rel := range ix.included {
		roots = append(roots, filepath.Join(ix.root, filepath.FromSlash(rel)))
	}
	return roots
}

// Skipped reports whether a watch of the root should leave out the folder
// at rel (root-relative, slash-separated).
func (ix *Index) Skipped(rel string) bool {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.skip(rel, pathBase(rel))
}

func (ix *Index) skip(rel, name string) bool {
	if ix.skipNames[name] {
		return true
	}
	for _, inc := range ix.included {
		if rel == inc || strings.HasPrefix(rel, inc+"/") {
			return false
		}
	}
	for _, p := range ix.skipPaths {
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

// Build indexes the root from scratch, then the included places.
func (ix *Index) Build(ctx context.Context) error {
	ix.mu.Lock()
	ix.entries, ix.names, ix.lower, ix.children, ix.spelled, ix.deleted, ix.included = nil, nil, nil, map[int32][]int32{}, map[int32][]syllable{}, 0, nil
	ix.add(-1, "", time.Time{}, flagDir|flagHidden)
	ix.mu.Unlock()
	ix.scan(ctx, 0, ix.root, "")
	for _, pattern := range ix.include {
		found, err := filepath.Glob(filepath.Join(ix.root, filepath.FromSlash(pattern)))
		if err != nil {
			return err
		}
		for _, abs := range found {
			if info, err := os.Lstat(abs); err != nil || !info.IsDir() {
				continue
			}
			rel, _ := filepath.Rel(ix.root, abs)
			rel = filepath.ToSlash(rel)
			ix.mu.Lock()
			ix.included = append(ix.included, rel)
			idx := ix.ensure(rel)
			ix.mu.Unlock()
			ix.scan(ctx, idx, abs, rel)
		}
	}
	return ctx.Err()
}

// ensure adds the hidden folders leading to rel and returns rel's entry.
// Call with ix.mu held.
func (ix *Index) ensure(rel string) int32 {
	idx := int32(0)
	for _, part := range strings.Split(rel, "/") {
		if c, ok := ix.child(idx, part); ok {
			idx = c
			continue
		}
		idx = ix.add(idx, part, time.Time{}, flagDir|flagHidden)
	}
	return idx
}

func (ix *Index) child(parent int32, name string) (int32, bool) {
	for _, c := range ix.children[parent] {
		if string(ix.nameOf(c)) == name {
			return c, true
		}
	}
	return 0, false
}

func (ix *Index) nameOf(i int32) []byte {
	e := &ix.entries[i]
	return ix.names[e.name : e.name+uint32(e.nameLen)]
}

func (ix *Index) lowerOf(i int32) []byte {
	e := &ix.entries[i]
	if e.flags&flagFolded == 0 {
		return ix.names[e.name : e.name+uint32(e.nameLen)]
	}
	return ix.lower[e.lower : e.lower+uint32(e.lowerLen)]
}

// add appends an entry. Call with ix.mu held.
func (ix *Index) add(parent int32, name string, mod time.Time, flags uint8) int32 {
	name = name[:min(len(name), 0xffff)]
	e := entry{parent: parent, name: uint32(len(ix.names)), nameLen: uint16(len(name)), flags: flags}
	if !mod.IsZero() {
		e.mod = uint32(max(mod.Unix(), 0))
	}
	ix.names = append(ix.names, name...)
	if lower := strings.ToLower(name); lower != name {
		lower = lower[:min(len(lower), 0xffff)]
		e.flags |= flagFolded
		e.lower, e.lowerLen = uint32(len(ix.lower)), uint16(len(lower))
		ix.lower = append(ix.lower, lower...)
	}
	idx := int32(len(ix.entries))
	if sp := spell(name); sp != nil {
		// a few percent of names; their syllables are kept beside the arenas
		e.flags |= flagSpelled
		ix.spelled[idx] = sp
	}
	ix.entries = append(ix.entries, e)
	if parent >= 0 {
		ix.children[parent] = append(ix.children[parent], idx)
	}
	return idx
}

// remove marks an entry and everything in it deleted. Call with ix.mu held.
func (ix *Index) remove(i int32) {
	e := &ix.entries[i]
	if e.flags&flagDeleted != 0 {
		return
	}
	e.flags |= flagDeleted
	ix.deleted++
	if siblings := ix.children[e.parent]; e.parent >= 0 {
		if k := slices.Index(siblings, i); k >= 0 {
			ix.children[e.parent] = slices.Delete(siblings, k, k+1)
		}
	}
	for _, c := range slices.Clone(ix.children[i]) {
		ix.remove(c)
	}
	delete(ix.children, i)
	delete(ix.spelled, i)
}

type item struct {
	name string
	mod  time.Time
	dir  bool
}

// list reads a folder's files and folders, leaving out symlinks and
// special files; a folder that can't be read lists nothing.
func list(abs string) ([]item, error) {
	entries, err := os.ReadDir(abs)
	items := make([]item, 0, len(entries))
	for _, d := range entries {
		kind := d.Type()
		if kind&fs.ModeSymlink != 0 || !kind.IsDir() && !kind.IsRegular() {
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		items = append(items, item{name: d.Name(), mod: info.ModTime(), dir: kind.IsDir()})
	}
	return items, err
}

type folder struct {
	idx      int32
	abs, rel string
	items    []item
}

// scan indexes everything in a folder the index has no children for yet,
// reading folders in parallel and adding them on this goroutine.
func (ix *Index) scan(ctx context.Context, idx int32, abs, rel string) {
	jobs, done := make(chan folder), make(chan folder)
	var workers sync.WaitGroup
	for range 2 * runtime.GOMAXPROCS(0) {
		workers.Go(func() {
			for f := range jobs {
				f.items, _ = list(f.abs)
				done <- f
			}
		})
	}
	queue, pending := []folder{{idx: idx, abs: abs, rel: rel}}, 0
	for (len(queue) > 0 || pending > 0) && ctx.Err() == nil {
		var send chan folder
		var next folder
		if len(queue) > 0 {
			send, next = jobs, queue[len(queue)-1]
		}
		select {
		case send <- next:
			queue = queue[:len(queue)-1]
			pending++
		case f := <-done:
			pending--
			ix.mu.Lock()
			for _, it := range f.items {
				childRel := join(f.rel, it.name)
				if it.dir && ix.skip(childRel, it.name) {
					continue
				}
				if it.dir {
					c := ix.add(f.idx, it.name, it.mod, flagDir)
					queue = append(queue, folder{idx: c, abs: filepath.Join(f.abs, it.name), rel: childRel})
				} else {
					ix.add(f.idx, it.name, it.mod, 0)
				}
			}
			ix.mu.Unlock()
		}
	}
	close(jobs)
	go func() {
		for range done { // let workers still sending after a cancel finish
		}
	}()
	workers.Wait()
	close(done)
}

func join(rel, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
}

func pathBase(rel string) string { return rel[strings.LastIndexByte(rel, '/')+1:] }

// Refresh rereads a folder a watch said changed: it adds and removes what
// changed in it, and rereads the folders in it whose modification time
// changed, as a folder's does when entries are added, removed or renamed.
// A folder the index doesn't know refreshes the nearest one it does.
func (ix *Index) Refresh(abs string) {
	rel, err := filepath.Rel(ix.root, filepath.Clean(abs))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return
	}
	ix.mu.RLock()
	idx, known := int32(0), ""
	if rel != "." {
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			c, ok := ix.child(idx, part)
			if !ok || ix.entries[c].flags&flagDir == 0 {
				break
			}
			idx, known = c, join(known, part)
		}
	}
	ix.mu.RUnlock()
	ix.refresh(idx, filepath.Join(ix.root, filepath.FromSlash(known)), known)
}

func (ix *Index) refresh(idx int32, abs, rel string) {
	items, err := list(abs)
	if os.IsNotExist(err) && idx != 0 {
		ix.mu.Lock()
		parent := ix.entries[idx].parent
		ix.mu.Unlock()
		ix.refresh(parent, filepath.Dir(abs), rel[:max(strings.LastIndexByte(rel, '/'), 0)])
		return
	}
	var fresh, changed []folder
	ix.mu.Lock()
	existing := map[string]int32{}
	for _, c := range ix.children[idx] {
		existing[string(ix.nameOf(c))] = c
	}
	for _, it := range items {
		childRel := join(rel, it.name)
		skipped := it.dir && ix.skip(childRel, it.name)
		if c, ok := existing[it.name]; ok {
			delete(existing, it.name)
			e := &ix.entries[c]
			hidden := e.flags&flagHidden != 0
			switch {
			case (e.flags&flagDir != 0) != it.dir || skipped && !hidden:
				ix.remove(c)
			default:
				mod := uint32(max(it.mod.Unix(), 0))
				if it.dir && mod != e.mod {
					changed = append(changed, folder{idx: c, abs: filepath.Join(abs, it.name), rel: childRel})
				}
				e.mod = mod
				continue
			}
		}
		if skipped {
			continue
		}
		flags := uint8(0)
		if it.dir {
			flags = flagDir
		}
		c := ix.add(idx, it.name, it.mod, flags)
		if it.dir {
			fresh = append(fresh, folder{idx: c, abs: filepath.Join(abs, it.name), rel: childRel})
		}
	}
	if err == nil {
		for _, c := range existing {
			ix.remove(c)
		}
	}
	ix.mu.Unlock()
	for _, f := range fresh {
		ix.scan(context.Background(), f.idx, f.abs, f.rel)
	}
	for _, f := range changed {
		ix.refresh(f.idx, f.abs, f.rel)
	}
}

type candidate struct {
	idx   int32
	score int
}

// Search returns the best matches for query, best first. The last word of
// the query matches names; any earlier words must each appear in the name
// of a folder above.
func (ix *Index) Search(query string, limit int) []Match {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 || limit <= 0 {
		return nil
	}
	q := []byte(words[len(words)-1])
	folders := make([][]byte, len(words)-1)
	for i, w := range words[:len(words)-1] {
		folders[i] = []byte(w)
	}
	now := time.Now().Unix()

	ix.mu.RLock()
	defer ix.mu.RUnlock()
	n := len(ix.entries)
	parts := runtime.GOMAXPROCS(0)
	best := make([][]candidate, parts)
	var wg sync.WaitGroup
	for p := range parts {
		wg.Go(func() {
			var top []candidate
			for i := int32(p * n / parts); i < int32((p+1)*n/parts); i++ {
				e := &ix.entries[i]
				if e.flags&(flagDeleted|flagHidden) != 0 {
					continue
				}
				s := ix.score(i, q)
				if s == 0 {
					continue
				}
				depth, hidden, ok := ix.above(i, folders)
				if !ok {
					continue
				}
				s += recency(now-int64(e.mod)) - min(depth, 10)
				if hidden {
					s -= hiddenPenalty
				}
				top = insert(top, candidate{i, s}, limit)
			}
			best[p] = top
		})
	}
	wg.Wait()
	var all []candidate
	for _, top := range best {
		for _, c := range top {
			all = insert(all, c, limit)
		}
	}
	matches := make([]Match, len(all))
	for k, c := range all {
		e := &ix.entries[c.idx]
		matches[k] = Match{Path: ix.path(c.idx), Name: string(ix.nameOf(c.idx)), Dir: e.flags&flagDir != 0, Modified: time.Unix(int64(e.mod), 0), Score: c.score}
	}
	return matches
}

// hiddenPenalty ranks what lies in hidden folders, mostly tool caches and
// copies of projects, below the user's own files without hiding it.
const hiddenPenalty = 150

// above reports how deep entry i is, whether a folder above it is hidden
// (its name starts with a dot), and whether each of words appears in the
// name of a folder above it.
func (ix *Index) above(i int32, words [][]byte) (depth int, hidden, ok bool) {
	found := 0
	var seen uint64
	for p := ix.entries[i].parent; p > 0; p = ix.entries[p].parent {
		depth++
		if name := ix.nameOf(p); len(name) > 0 && name[0] == '.' {
			hidden = true
		}
		for k, w := range words {
			if k < 64 && seen&(1<<k) == 0 && ix.contains(p, w) {
				seen |= 1 << k
				found++
			}
		}
	}
	return depth, hidden, found == len(words)
}

func (ix *Index) score(i int32, q []byte) int {
	if ix.entries[i].flags&flagSpelled == 0 {
		return Score(q, ix.lowerOf(i), ix.nameOf(i))
	}
	return scoreSpelled(q, ix.lowerOf(i), ix.nameOf(i), ix.spelled[i])
}

// contains reports whether entry i's name contains w, or w spells part of it in pinyin.
func (ix *Index) contains(i int32, w []byte) bool {
	if bytes.Contains(ix.lowerOf(i), w) {
		return true
	}
	return ix.entries[i].flags&flagSpelled != 0 && scorePinyin(string(w), ix.spelled[i]) > 0
}

func recency(age int64) int {
	switch {
	case age < 24*3600:
		return 30
	case age < 7*24*3600:
		return 20
	case age < 30*24*3600:
		return 10
	}
	return 0
}

// insert keeps the limit best candidates, best first; equal scores keep the
// earlier entry first.
func insert(top []candidate, c candidate, limit int) []candidate {
	k := len(top)
	for k > 0 && (top[k-1].score < c.score || top[k-1].score == c.score && top[k-1].idx > c.idx) {
		k--
	}
	if k >= limit {
		return top
	}
	top = slices.Insert(top, k, c)
	return top[:min(len(top), limit)]
}

func (ix *Index) path(i int32) string {
	var parts []string
	for ; i > 0; i = ix.entries[i].parent {
		parts = append(parts, string(ix.nameOf(i)))
	}
	slices.Reverse(parts)
	return filepath.Join(append([]string{ix.root}, parts...)...)
}
