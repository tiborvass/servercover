package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

func fatal(arg interface{}) {
	panic(arg)
	log.Fatal(arg)
}

func fatalf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
	log.Fatalf(format, args...)
}

var errMandatoryImports = errors.New("Please import net, net/rpc and testing packages")

var socket = flag.String("socket", "", "unix socket on which to exchange coverage information")
var coverpkg = flag.String("coverpkg", "", "See 'go help test'")
var output = flag.String("o", "", "output filename")
var v = flag.Bool("v", false, "verbose")
var work = flag.Bool("work", false, "work")

var testCoverPaths []string

type coverVar struct {
	File string
	Var  string
}

type coverInfo struct {
	ImportPath  string
	PackageName string
	Vars        map[string]*coverVar
}

type coverage struct {
	Cover     []coverInfo
	CoverMode string
	Socket    string
}

func (c *coverage) Covered() string {
	if testCoverPaths == nil {
		return ""
	}
	return " in " + strings.Join(testCoverPaths, ", ")
}

func writeFile(out string, c *coverage) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	return testmainTmpl.Execute(f, c)
}

func main() {
	//test.CmdTest.Run(nil, os.Args[1:])
	//return
	flag.Parse()
	if *socket == "" {
		fatalf("need to specify -socket")
	}
	if *coverpkg == "" {
		*coverpkg = strings.Join(flag.Args(), ",")
	}
	gopath, err := exec.Command("go", "env", "GOPATH").CombinedOutput()
	if err != nil {
		fatal(err)
	}
	if *v {
		fmt.Printf("GOPATH=%s\n", gopath)
	}

	var gofiles []string

	cmdargs := []string{"test", "-c", "-a", "-work", "-covermode=set", "-coverpkg", *coverpkg}
	cmdargs = append(cmdargs, flag.Args()...)
	cmd := exec.Command("go", cmdargs...)
	if *v {
		fmt.Println(strings.Join(cmd.Args, " "))
	}
	envstr, err := cmd.CombinedOutput()
	if err != nil {
		fatal(string(envstr))
	}
	if *v {
		fmt.Println("output:", string(envstr))
	}
	workPrefix := []byte("WORK=")
	if !bytes.HasPrefix(envstr, workPrefix) {
		fatalf("Expected output to have %s prefix, instead got:\n%s", workPrefix, envstr)
	}
	newline := bytes.Index(envstr, []byte{'\n'})
	if newline < 0 {
		newline = len(envstr)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	workdir := string(envstr[len(workPrefix):newline])
	fmt.Printf("WORK=%s\n", workdir)

	c := &coverage{CoverMode: "set", Socket: *socket}

	if err := filepath.Walk(workdir, func(path string, info os.FileInfo, err error) error {
		if path == workdir {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		vars := map[string]*coverVar{}
		var packageName, importPath *string
		if err := filepath.Walk(path, func(fpath string, info os.FileInfo, err error) error {
			if path == fpath {
				return nil
			}
			if err != nil {
				return err
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			if !strings.HasSuffix(fpath, ".cover.go") {
				return nil
			}
			f, err := os.Open(fpath)
			if err != nil {
				return err
			}
			defer f.Close()
			s := bufio.NewScanner(f)
			var file *string
			for s.Scan() {
				t := s.Text()
				if file == nil {
					t = t[7 : len(t)-2]
					t = t[len(filepath.Join(string(gopath), "src")):]
					file = &t
					tt := t[:strings.LastIndex(t, "/")]
					importPath = &tt
					continue
				}
				if strings.HasPrefix(t, "package ") {
					t = t[len("package "):]
					if i := strings.Index(t, " "); i >= 0 {
						t = t[:i]
					}
					if packageName == nil {
						packageName = &t
					} else if *packageName != t {
						return fmt.Errorf("Found package %q, expected %q", t, *packageName)
					}
					continue
				}
				if strings.HasPrefix(t, "var GoCover_") && strings.HasSuffix(t, " = struct {") {
					t = t[4:] // remove "var " to start with "GoCover"
					vars[*file] = &coverVar{File: *file, Var: t[:strings.Index(t, " ")]}
					break
				}
			}
			if *packageName == "main" {
				gofiles = append(gofiles, fpath)
			}
			return s.Err()
		}); err != nil {
			return err
		}
		if packageName == nil && importPath == nil {
			return nil
		}
		if packageName == nil || importPath == nil {
			return fmt.Errorf("Could not find package or importPath")
		}
		c.Cover = append(c.Cover, coverInfo{PackageName: *packageName, ImportPath: *importPath, Vars: vars})
		return nil
	}); err != nil {
		fatal(err)
	}

	maincoverFile := filepath.Join(workdir, "b001", "000_maincover.go")
	gofiles = append(gofiles, maincoverFile)

	if err := writeFile(maincoverFile, c); err != nil {
		fatal(err)
	}

	importcfgMap := map[string]struct{}{}

	if err := filepath.Walk(workdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		f, err := os.Open(filepath.Join(path, "importcfg"))
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		for s.Scan() {
			t := s.Text()
			if t[0] == '#' || !strings.HasPrefix(t, "packagefile ") {
				continue
			}
			importcfgMap[t] = struct{}{}
		}
		return s.Err()
	}); err != nil {
		fatal(err)
	}

	var foundNet, foundNetRpc, foundTesting bool
	importcfgs := make([]string, 0, len(importcfgMap))
	for s := range importcfgMap {
		if strings.HasPrefix(s, "packagefile net=") {
			foundNet = true
		}
		if strings.HasPrefix(s, "packagefile net/rpc=") {
			foundNetRpc = true
		}
		if strings.HasPrefix(s, "packagefile testing=") {
			foundTesting = true
		}
		importcfgs = append(importcfgs, s)
	}
	if !foundNet || !foundNetRpc || !foundTesting {
		fatal(errMandatoryImports)
	}
	sort.Strings(importcfgs)

	importcfgPath := filepath.Join(workdir, "importcfg")
	f, err := os.Create(importcfgPath)
	if err != nil {
		fatal(err)
	}
	for _, cfg := range importcfgs {
		f.Write([]byte(cfg))
		f.Write([]byte{'\n'})
	}
	f.Close()

	if *output == "" {
		*output = filepath.Join(cwd, filepath.Base(cwd))
	}

	cmdArgs := []string{"tool", "compile", "-o", filepath.Join(workdir, "b001", "_pkg_.a"), "-importcfg", filepath.Join(workdir, "b001", "importcfg")}
	cmdArgs = append(cmdArgs, gofiles...)
	out, err := exec.Command("go", cmdArgs...).CombinedOutput()
	if err != nil {
		fatal(string(out))
	}

	//defer os.Chdir(cwd)
	//os.Chdir(workdir)
	cmd = exec.Command("go", "tool", "link", "-o", *output, "-importcfg", importcfgPath, "-buildmode=exe", filepath.Join(workdir, "b001", "_pkg_.a"))
	if *v {
		fmt.Println(strings.Join(cmd.Args, " "))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

var testmainTmpl = template.Must(template.New("main").Parse(`
package main

import (
	"net"
	"net/rpc"
	"testing"

{{range $i, $p := .Cover}}{{if ne $p.PackageName "main"}}
	_cover{{$i}} {{$p.ImportPath | printf "%q"}}
{{end}}{{end}}
)

type ServerCover testing.Cover

func (mc *ServerCover) GetCover(shouldResetCover bool, reply *testing.Cover) error {
	*reply = testing.Cover(*mc)
	return nil
}

func init() {
	// Only updated by init functions, so no need for atomicity.
	var serverCover = &ServerCover{
		Mode: {{printf "%q" .CoverMode}},
		Counters: make(map[string][]uint32),
		Blocks: make(map[string][]testing.CoverBlock),
		CoveredPackages: {{printf "%q" .Covered}},
	}


	coverRegisterFile := func(fileName string, counter []uint32, pos []uint32, numStmts []uint16) {
		if 3*len(counter) != len(pos) || len(counter) != len(numStmts) {
			panic("coverage: mismatched sizes")
		}
		if serverCover.Counters[fileName] != nil {
			// Already registered.
			return
		}
		serverCover.Counters[fileName] = counter
		block := make([]testing.CoverBlock, len(counter))
		for i := range counter {
			block[i] = testing.CoverBlock{
				Line0: pos[3*i+0],
				Col0: uint16(pos[3*i+2]),
				Line1: pos[3*i+1],
				Col1: uint16(pos[3*i+2]>>16),
				Stmts: numStmts[i],
			}
		}
		serverCover.Blocks[fileName] = block
	}
	{{range $i, $p := .Cover}}
	{{range $file, $cover := $p.Vars}}
	coverRegisterFile({{printf "%q" $cover.File}}, {{if ne $p.PackageName "main"}}_cover{{$i}}.{{end}}{{$cover.Var}}.Count[:], {{if ne $p.PackageName "main"}}_cover{{$i}}.{{end}}{{$cover.Var}}.Pos[:], {{if ne $p.PackageName "main"}}_cover{{$i}}.{{end}}{{$cover.Var}}.NumStmt[:])
	{{end}}
	{{end}}
	rpc.Register(serverCover)
	go func() {
		ln, err := net.Listen("unix", {{printf "%q" .Socket}})
		if err != nil {
			panic(err)
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}
			go rpc.ServeConn(conn)
		}
	}()
}
`))
