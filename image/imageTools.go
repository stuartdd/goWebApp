package image

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type PicFile struct {
	N string
	D string
}

type PicDir struct {
	Dir   string
	Files []*PicFile
	Subs  []*PicDir
}

type PicPath struct {
	paths []string
}

func newPicPath() *PicPath {
	return &PicPath{
		paths: []string{},
	}
}

func (p *PicPath) String() string {
	var line bytes.Buffer
	for _, pa := range p.paths {
		line.WriteString(pa)
		line.WriteRune('/')
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

func (p *PicDir) VisitEachFile(onFile func(*PicPath, string)) {
	pp := newPicPath()
	p.visitEachFile(pp, onFile)
}

func (p *PicDir) visitEachFile(path *PicPath, onFile func(*PicPath, string)) {
	for _, f := range p.Files {
		if onFile != nil {
			onFile(path, f.N)
		}
	}
	for _, d := range p.Subs {
		path.push(d.Dir)
		d.visitEachFile(path, onFile)
		path.pop()
	}
}

func newPicDir(dir string) *PicDir {
	return &PicDir{
		Dir:   dir,
		Files: []*PicFile{},
		Subs:  []*PicDir{},
	}
}

func (p *PicDir) Add(path string) {
	p.addParts(strings.Split(path, "/"))
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
				p.Subs = append(p.Subs, sub)
			}
			sub.addParts(parts[1:])
		}
	}
}

func (p *PicDir) hasSub(name string) *PicDir {
	for _, sub := range p.Subs {
		if sub.Dir == name {
			return sub
		}
	}
	return nil
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
	dir := newPicDir("root")
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
