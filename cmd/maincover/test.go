package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var coverpkg = flag.String("coverpkg", "", "See 'go help test'")
var output = flag.String("o", "", "output filename")
var v = flag.Bool("v", false, "verbose")

type coverVar struct {
	File string
	Var  string
}

type coverInfo map[string]*coverVar

type coverage struct {
	Cover []coverInfo
}

func main() {
	flag.Parse()
	if *coverpkg == "" {
		*coverpkg = strings.Join(flag.Args(), ",")
	}
	cmdargs := []string{"test", "-v", "-x", "-c", "-a", "-work", "-covermode=set", "-coverpkg", *coverpkg}
	cmdargs = append(cmdargs, flag.Args()...)
	cmd := exec.Command("go", cmdargs...)
	if *v {
		fmt.Println(strings.Join(cmd.Args, " "))
	}
	envstr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	if *v {
		fmt.Println("output:", string(envstr))
	}
	workPrefix := []byte("WORK=")
	if !bytes.HasPrefix(envstr, workPrefix) {
		log.Fatalf("Expected output to have %s prefix, instead got:\n%s", workPrefix, envstr)
	}
	newline := bytes.Index(envstr, []byte{'\n'})
	if newline < 0 {
		newline = len(envstr)
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	work := string(envstr[len(workPrefix):newline])

	c := &coverage{
		Cover: []coverInfo{
			{"program.go": &coverVar{}},
		},
	}

	if err := writeFile(filepath.Join(work, "b001", "000_maincover.go"), c); err != nil {
		log.Fatal(err)
	}
	defer os.Chdir(cwd)
	os.Chdir(filepath.Join(work, "b001"))
	if *output == "" {
		*output = filepath.Join(cwd, filepath.Base(cwd))
	}
	cmd = exec.Command("go", "build", "-o", *output)
	if *v {
		fmt.Println(strings.Join(cmd.Args, " "))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	/*
		//os.Setenv("WORK", work)
	*/

	/*
		fmt.Println(os.Args[1:])
		m := buildutil.ExpandPatterns(&build.Default, os.Args[1:])
		fmt.Println(m)
		var conf loader.Config
		_, err := conf.FromArgs(os.Args[1:], false)
		if err != nil {
			log.Fatal(err)
		}
		p, err := conf.Load()
		if err != nil {
			log.Fatal(err)
		}
		for _, pkg := range p.AllPackages {
			fmt.Printf("%s\t\t%#v\n", pkg.Pkg.Path(), pkg.Pkg)
		}
	*/
}

func writeFile(fileName string, c *coverage) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	return maincoverTmpl.Execute(f, c)
}

var maincoverTmpl = template.Must(template.New("main").Parse(`package main

import (
	"fmt"
	"net/rpc"
	"testing"
	"strings"
)

type MainCover struct {
	counters map[string][]uint32
	blocks   map[string][]testing.CoverBlock
	covered  []string
}

func (m *MainCover) GetCover(shouldResetCover bool, c *testing.Cover) error {
	coveredStr := ""
	if len(m.covered) == 0 {
		coveredStr = " in " + strings.Join(m.covered, ", ")
	}
	*c = testing.Cover{
		Mode: "",
		Counters: m.counters,
		Blocks: m.blocks,
		CoveredPackages: coveredStr,
	}
	return nil
}

func init() {
	fmt.Println("yolo")
	mainCover := &MainCover{
		counters: make(map[string][]uint32),
		blocks: make(map[string][]testing.CoverBlock),
		covered: nil,
	}
	{{range $i, $p := .Cover}}
	{{range $file, $cover := $p.Vars}}
	coverRegisterFile({{printf "%q" $cover.File}}, _cover{{$i}}.{{$cover.Var}}.Count[:], _cover{{$i}}.{{$cover.Var}}.Pos[:], _cover{{$i}}.{{$cover.Var}}.NumStmt[:])
	{{end}}
	{{end}}
	rpc.Register(mainCover)
	
}
`))

// //go:linkname testmainTmpl cmd/go/internal/test.testmainTmpl
// var testmainTmpl *template.Template

var testmainTmpl = template.Must(template.New("main2").Parse(`
package main

import (
{{if not .TestMain}}
	"os"
{{end}}
	"testing"
	"testing/internal/testdeps"

{{if .ImportTest}}
	{{if .NeedTest}}_test{{else}}_{{end}} {{.Package.ImportPath | printf "%q"}}
{{end}}
{{if .ImportXtest}}
	{{if .NeedXtest}}_xtest{{else}}_{{end}} {{.Package.ImportPath | printf "%s_test" | printf "%q"}}
{{end}}
{{range $i, $p := .Cover}}
	_cover{{$i}} {{$p.Package.ImportPath | printf "%q"}}
{{end}}
)

var tests = []testing.InternalTest{
{{range .Tests}}
	{"{{.Name}}", {{.Package}}.{{.Name}}},
{{end}}
}

var benchmarks = []testing.InternalBenchmark{
{{range .Benchmarks}}
	{"{{.Name}}", {{.Package}}.{{.Name}}},
{{end}}
}

var examples = []testing.InternalExample{
{{range .Examples}}
	{"{{.Name}}", {{.Package}}.{{.Name}}, {{.Output | printf "%q"}}, {{.Unordered}}},
{{end}}
}

func init() {
	testdeps.ImportPath = {{.ImportPath | printf "%q"}}
}

{{if .CoverEnabled}}

// Only updated by init functions, so no need for atomicity.
var (
	coverCounters = make(map[string][]uint32)
	coverBlocks = make(map[string][]testing.CoverBlock)
)

func init() {
	{{range $i, $p := .Cover}}
	{{range $file, $cover := $p.Vars}}
	coverRegisterFile({{printf "%q" $cover.File}}, _cover{{$i}}.{{$cover.Var}}.Count[:], _cover{{$i}}.{{$cover.Var}}.Pos[:], _cover{{$i}}.{{$cover.Var}}.NumStmt[:])
	{{end}}
	{{end}}
}

func coverRegisterFile(fileName string, counter []uint32, pos []uint32, numStmts []uint16) {
	if 3*len(counter) != len(pos) || len(counter) != len(numStmts) {
		panic("coverage: mismatched sizes")
	}
	if coverCounters[fileName] != nil {
		// Already registered.
		return
	}
	coverCounters[fileName] = counter
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
	coverBlocks[fileName] = block
}
{{end}}

func main() {
{{if .CoverEnabled}}
	testing.RegisterCover(testing.Cover{
		Mode: {{printf "%q" .CoverMode}},
		Counters: coverCounters,
		Blocks: coverBlocks,
		CoveredPackages: {{printf "%q" .Covered}},
	})
{{end}}
	m := testing.MainStart(testdeps.TestDeps{}, tests, benchmarks, examples)
{{with .TestMain}}
	{{.Package}}.{{.Name}}(m)
{{else}}
	os.Exit(m.Run())
{{end}}
}

`))
