package sessions

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Dirs says where to read from and where to keep the cache.
type Dirs struct {
	Projects string // ~/.claude/projects
	Desktop  string // ~/Library/Application Support/Claude/claude-code-sessions
	Cache    string // the cache file; "" keeps the cache in memory only
}

// DefaultDirs are this user's folders. CLAUDE_CONFIG_DIR moves ~/.claude, as it does for Claude Code.
func DefaultDirs() (Dirs, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Dirs{}, err
	}
	claudeDir := filepath.Join(home, ".claude")
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		claudeDir = v
	}
	conf, err := os.UserConfigDir()
	if err != nil {
		return Dirs{}, err
	}
	return Dirs{
		Projects: filepath.Join(claudeDir, "projects"),
		Desktop:  filepath.Join(conf, "Claude", "claude-code-sessions"),
		Cache:    filepath.Join(conf, "ClaudeSessions", "index.json"),
	}, nil
}

// cacheVersion changes whenever facts or desktopInfo change shape, so an old cache file is read again from scratch.
const cacheVersion = 1

type stamp struct {
	Size    int64 `json:"size"`
	ModTime int64 `json:"mtime"` // nanoseconds
}

type cachedFacts struct {
	Stamp stamp `json:"stamp"`
	Facts facts `json:"facts"`
}

type cachedDesktop struct {
	Stamp stamp       `json:"stamp"`
	Info  desktopInfo `json:"info"`
}

type cacheFile struct {
	Version     int                      `json:"version"`
	Transcripts map[string]cachedFacts   `json:"transcripts"`
	Desktop     map[string]cachedDesktop `json:"desktop"`
}

// Listing is every group, with the counts the window shows at the top.
type Listing struct {
	Groups      []Group   `json:"groups"`
	Sessions    int       `json:"sessions"`
	Transcripts int       `json:"transcripts"`
	ReadFiles   int       `json:"readFiles"` // files read again this time, not taken from the cache
	Problems    []string  `json:"problems"`
	ScannedAt   time.Time `json:"scannedAt"`
	Took        string    `json:"took"`
}

// Index lists the sessions on this machine. It reads only the files that changed since the last look.
type Index struct {
	dirs Dirs

	mu     sync.Mutex
	cache  cacheFile
	loaded bool
	paths  map[string]string // session id → transcript path
}

func NewIndex(d Dirs) *Index {
	return &Index{dirs: d, paths: map[string]string{},
		cache: cacheFile{Version: cacheVersion, Transcripts: map[string]cachedFacts{}, Desktop: map[string]cachedDesktop{}}}
}

func stampOf(fi fs.FileInfo) stamp { return stamp{Size: fi.Size(), ModTime: fi.ModTime().UnixNano()} }

type found struct {
	path  string
	stamp stamp
}

// transcriptFiles is every <projects>/<folder>/<id>.jsonl; subagent and tool-result files live deeper and are skipped.
func transcriptFiles(dir string) ([]found, error) {
	folders, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []found
	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				continue
			}
			out = append(out, found{filepath.Join(dir, f.Name(), e.Name()), stampOf(fi)})
		}
	}
	return out, nil
}

// desktopFiles is every local_*.json under the desktop app's session folder, at any depth.
func desktopFiles(dir string) []found {
	var out []found
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasPrefix(d.Name(), "local_") || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			out = append(out, found{p, stampOf(fi)})
		}
		return nil
	})
	return out
}

// eachInParallel runs fn on every item with one worker per CPU core, at most 8.
func eachInParallel(items []found, fn func(found)) {
	workers := min(runtime.NumCPU(), 8)
	ch := make(chan found)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range ch {
				fn(it)
			}
		}()
	}
	for _, it := range items {
		ch <- it
	}
	close(ch)
	wg.Wait()
}

func (x *Index) loadCache() {
	if x.loaded {
		return
	}
	x.loaded = true
	if x.dirs.Cache == "" {
		return
	}
	b, err := os.ReadFile(x.dirs.Cache)
	if err != nil {
		return
	}
	var c cacheFile
	if json.Unmarshal(b, &c) != nil || c.Version != cacheVersion || c.Transcripts == nil || c.Desktop == nil {
		return
	}
	x.cache = c
}

func (x *Index) saveCache() error {
	if x.dirs.Cache == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(x.dirs.Cache), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(x.cache)
	if err != nil {
		return err
	}
	tmp := x.dirs.Cache + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, x.dirs.Cache)
}

// List reads what changed and returns every session, grouped by folder.
func (x *Index) List() (Listing, error) {
	start := time.Now()
	x.mu.Lock()
	defer x.mu.Unlock()
	x.loadCache()

	out := Listing{Groups: []Group{}, Problems: []string{}}
	tfiles, err := transcriptFiles(x.dirs.Projects)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return out, err
	}
	dfiles := desktopFiles(x.dirs.Desktop)

	var pmu sync.Mutex
	problem := func(s string) { pmu.Lock(); out.Problems = append(out.Problems, s); pmu.Unlock() }

	// Transcripts: read again only the ones whose size or time changed.
	var stale []found
	nextT := make(map[string]cachedFacts, len(tfiles))
	for _, f := range tfiles {
		if c, ok := x.cache.Transcripts[f.path]; ok && c.Stamp == f.stamp {
			nextT[f.path] = c
		} else {
			stale = append(stale, f)
		}
	}
	var tmu sync.Mutex
	eachInParallel(stale, func(f found) {
		fa, err := scanFile(f.path)
		if err != nil {
			problem(f.path + ": " + err.Error()) // what was read before the error is still kept
		}
		tmu.Lock()
		nextT[f.path] = cachedFacts{Stamp: f.stamp, Facts: fa}
		tmu.Unlock()
	})
	out.ReadFiles = len(stale)

	// Desktop records, the same way.
	stale = stale[:0]
	nextD := make(map[string]cachedDesktop, len(dfiles))
	for _, f := range dfiles {
		if c, ok := x.cache.Desktop[f.path]; ok && c.Stamp == f.stamp {
			nextD[f.path] = c
		} else {
			stale = append(stale, f)
		}
	}
	var dmu sync.Mutex
	eachInParallel(stale, func(f found) {
		info, err := readDesktop(f.path)
		if err != nil {
			problem(f.path + ": " + err.Error())
			return
		}
		dmu.Lock()
		nextD[f.path] = cachedDesktop{Stamp: f.stamp, Info: info}
		dmu.Unlock()
	})
	out.ReadFiles += len(stale)

	x.cache.Transcripts, x.cache.Desktop = nextT, nextD
	if out.ReadFiles > 0 {
		if err := x.saveCache(); err != nil {
			out.Problems = append(out.Problems, "cache: "+err.Error())
		}
	}

	// Join: one session per transcript, plus desktop sessions whose transcript is not on this machine.
	byCli := map[string]*desktopInfo{}
	for _, c := range nextD {
		info := c.Info
		if info.CliSessionID != "" {
			byCli[info.CliSessionID] = &info
		}
	}
	x.paths = map[string]string{}
	all := make([]Summary, 0, len(nextT)+len(byCli))
	seen := map[string]bool{}
	for p, c := range nextT {
		id := strings.TrimSuffix(filepath.Base(p), ".jsonl")
		if seen[id] {
			continue // the same id in two folders: keep the first; it is one session
		}
		seen[id] = true
		x.paths[id] = p
		fa := c.Facts
		all = append(all, summarize(id, &fa, byCli[id]))
	}
	out.Transcripts = len(all)
	for _, c := range nextD {
		info := c.Info
		if info.CliSessionID != "" && seen[info.CliSessionID] {
			continue
		}
		id := info.CliSessionID
		if id == "" {
			id = info.LocalID
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		all = append(all, summarize(id, nil, &info))
	}
	out.Sessions = len(all)
	out.Groups = buildGroups(all)
	out.ScannedAt = time.Now().UTC()
	out.Took = time.Since(start).Round(time.Millisecond).String()
	return out, nil
}

// TranscriptPath is the file of a session seen by the last List.
func (x *Index) TranscriptPath(id string) (string, bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	p, ok := x.paths[id]
	return p, ok
}
