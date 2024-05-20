package pictures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type ScannedData struct {
	OldState          *PicDir
	OldStateCount     int
	NewState          *PicDir
	NewStateCount     int
	NeedToCreate      *PicDir
	NeedToCreateCount int
	NeedToDelete      *PicDir
	NeedToDeleteCount int
}

type PicFile struct {
	N string
	D string
}

type PicDir struct {
	N     string
	Files []*PicFile
	Dirs  []*PicDir
}

type PicPath struct {
	paths []string
}

func newPicPath() *PicPath {
	return &PicPath{
		paths: []string{},
	}
}

func newPicPathFromFile(s string) *PicPath {
	if strings.TrimSpace(s) == "" {
		return newPicPath()
	}
	sList := strings.Split(s, "/")
	if len(sList) == 0 {
		return newPicPath()
	}
	if sList[0] == "" {
		return &PicPath{
			paths: sList[1:],
		}
	}
	return &PicPath{
		paths: sList,
	}
}

func (p *PicPath) Len() int {
	return len(p.paths)
}

func (p *PicPath) Last() string {
	return p.paths[len(p.paths)-1]
}
func (p *PicPath) Equal(pp *PicPath) bool {
	if p.Len() != pp.Len() {
		return false
	}
	for i, n := range p.paths {
		if pp.paths[i] != n {
			return false
		}
	}
	return true
}
func (p *PicPath) String() string {
	var line bytes.Buffer
	if p.Len() == 0 {
		return ""
	}
	for i, pa := range p.paths {
		line.WriteString(pa)
		if i < (len(p.paths) - 1) {
			line.WriteRune('/')
		}
	}
	return line.String()
}

func (p *PicPath) push(dir string) {
	p.paths = append(p.paths, dir)
}

func (p *PicPath) pop() string {
	if len(p.paths) > 0 {
		lp := p.paths[len(p.paths)-1]
		p.paths = p.paths[0 : len(p.paths)-1]
		return lp
	}
	return ""
}

func newPicDir(dir string) *PicDir {
	return &PicDir{
		N:     dir,
		Files: []*PicFile{},
		Dirs:  []*PicDir{},
	}
}

func (p *PicDir) Find(path *PicPath) (*PicDir, *PicFile) {
	l := path.Len()
	if l == 0 {
		return nil, nil
	}
	pp := p
	pos := 0
	var dir *PicDir
	if l > 1 {
		for pos = 0; pos < (l - 1); pos++ {
			dir = pp.FindDir(path.paths[pos])
			if dir == nil {
				return nil, nil
			}
			pp = dir
		}
	}
	return pp, pp.FindFile(path.Last())
}

func (p *PicDir) FindDir(N string) *PicDir {
	for _, d := range p.Dirs {
		if d.N == N {
			return d
		}
	}
	return nil
}

func (p *PicDir) FindFile(N string) *PicFile {
	for _, f := range p.Files {
		if f.N == N {
			return f
		}
	}
	return nil
}

func (p *PicDir) Load(fil string) (*PicDir, error) {
	dd, err := os.ReadFile(fil)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(dd, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PicDir) Save(fil string, indent bool) error {
	jData, err := p.toJson(indent)
	if err != nil {
		return err
	}
	return os.WriteFile(fil, jData, 0644)
}

func (p *PicDir) toJson(indent bool) ([]byte, error) {
	var j []byte
	var err error
	if indent {
		j, err = json.MarshalIndent(p, "", "  ")
	} else {
		j, err = json.Marshal(p)
	}
	if err != nil {
		return []byte{}, err
	}
	return j, nil
}

func (p *PicDir) VisitEachFile(onFile func(*PicPath, string) bool) {
	pp := newPicPath()
	cont := true
	p.visitEachFile(pp, &cont, onFile)
}

func (p *PicDir) visitEachFile(path *PicPath, cont *bool, onFile func(*PicPath, string) bool) {
	if *cont {
		for _, f := range p.Files {
			if onFile != nil {
				x := onFile(path, f.N)
				if !x {
					*cont = false
					return
				}
			}
		}
		for _, d := range p.Dirs {
			path.push(d.N)
			d.visitEachFile(path, cont, onFile)
			path.pop()
			if !*cont {
				return
			}

		}
	}
}

func (p *PicDir) VisitEachDir(onDir func(*PicPath, []*PicFile)) {
	pp := newPicPath()
	p.visitEachDir(pp, p.Files, onDir)
}

func (p *PicDir) visitEachDir(path *PicPath, files []*PicFile, onDir func(*PicPath, []*PicFile)) {
	if onDir != nil {
		onDir(path, files)
	}
	for _, d := range p.Dirs {
		path.push(d.N)
		d.visitEachDir(path, d.Files, onDir)
		path.pop()
	}
}

func (p *PicDir) Add(path string) {
	p.addParts(strings.Split(path, "/"))
}

func (p *PicDir) AddPath(path *PicPath) {
	p.addParts(path.paths)
}

func (p *PicDir) addParts(parts []string) {
	l := len(parts)
	if l > 0 {
		p0 := parts[0]
		if l == 1 {
			p.Files = append(p.Files, &PicFile{N: p0})
		} else {
			sub := p.hasSub(p0)
			if sub == nil {
				sub = newPicDir(p0)
				p.Dirs = append(p.Dirs, sub)
			}
			sub.addParts(parts[1:])
		}
	}
}

func (p *PicDir) hasSub(name string) *PicDir {
	for _, sub := range p.Dirs {
		if sub.N == name {
			return sub
		}
	}
	return nil
}

func InAnotB(a *PicDir, b *PicDir, notInB func(*PicPath)) {
	if notInB == nil {
		panic("InAnotB requires a callback function!")
	}
	a.VisitEachFile(func(pp *PicPath, s string) bool {
		path := newPicPathFromFile(fmt.Sprintf("%s/%s", pp.String(), s))
		dir, fil := b.Find(path)
		if dir == nil {
			notInB(path)
		} else {
			if fil == nil {
				notInB(path)
			}
		}
		return true
	})
}

func WalkDir(file string, onFile func(string, string) bool) (*PicDir, error) {
	f, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(f)
	if err != nil {
		return nil, err
	}
	pref := len(f) + 1
	dir := newPicDir(f)
	filepath.Walk(f, func(path string, info fs.FileInfo, err error) error {
		if info != nil {
			if !info.IsDir() {
				add := true
				if onFile != nil {
					add = onFile(path, info.Name())
				}
				if add {
					dir.Add(path[pref:])
				}
			}
		}
		return nil
	})
	return dir, nil
}

func ScanDirectory(dir string, ext []string) (*ScannedData, error) {
	dataDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(dataDir)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dataDir)
	}

	dataFile := filepath.Join(dataDir, "dirScanData.json")
	stat, err = os.Stat(dataFile)
	if err != nil {
		firstData, count, err := createScanData(dataDir, ext)
		if err != nil {
			return nil, err
		}
		err = firstData.Save(dataFile, true)
		if err != nil {
			return nil, err
		}
		return &ScannedData{
			OldState:          firstData,
			OldStateCount:     count,
			NewState:          nil,
			NewStateCount:     0,
			NeedToCreate:      nil,
			NeedToCreateCount: 0,
			NeedToDelete:      nil,
			NeedToDeleteCount: 0,
		}, nil
	} else {
		if stat.IsDir() {
			return nil, fmt.Errorf("%s is a directory", dataFile)
		}
	}

	oldData, err := newPicDir("").Load(dataFile)
	if err != nil {
		return nil, err
	}
	newData, count, err := createScanData(dataDir, ext)
	if err != nil {
		return nil, err
	}

	result := &ScannedData{
		OldState:          oldData,
		OldStateCount:     0,
		NewState:          newData,
		NewStateCount:     count,
		NeedToCreate:      newPicDir("Create"),
		NeedToCreateCount: 0,
		NeedToDelete:      newPicDir("Delete"),
		NeedToDeleteCount: 0,
	}
	result.compare()
	return result, nil
}

func (p *ScannedData) compare() {
	InAnotB(p.OldState, p.NewState, func(pp *PicPath) {
		p.NeedToDelete.AddPath(pp)
		p.NeedToDeleteCount++
	})
	InAnotB(p.NewState, p.OldState, func(pp *PicPath) {
		p.NeedToCreate.AddPath(pp)
		p.NeedToCreateCount++
	})
}

func createScanData(dir string, ext []string) (*PicDir, int, error) {
	count := 0
	sd, err := WalkDir(dir, func(p string, n string) bool {
		if n == "dirScanData.json" {
			return false // dont include the data file in the data
		}
		for _, ex := range ext {
			if !strings.HasSuffix(n, ex) {
				return false
			}
		}
		count++
		return true
	})
	if err != nil {
		return nil, 0, err
	}
	return sd, count, nil
}
