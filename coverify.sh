#!/bin/bash

# Coverify all packages under the current directory
# There needs to be a main package with a main() function somewhere or else an error is returned
# The main() function is renamed to __main(), and a new main() function is added that registers all the cover counters
# before calling __main(), after which all those cover counters are Gob-encoded to a Unix socket.
# This assumes there will be a listening Unix socket at the time the coverified binary is run.
coverify_main() {
	: ${mode:=set}

	WORK=$(mktemp -d -t go-maincover)
	[ -n "$DEBUG" ] && >&2 echo "WORK=$WORK"
	SRC="$WORK/src"
	IFS=$'\n'
	i=0
	import=()
	init=()
	for x in $(go list -f '{{with $c := .}}{{range .GoFiles}}{{printf "%s %s/%s\n" $c.ImportPath $c.Dir .}}{{end}}{{end}}' ./...); do
		f=$(echo "$x" | cut -d' ' -f2-)
		importpath=$(echo "$x" | cut -d' ' -f1)
		mkdir -p "$SRC/$importpath"
		cover="$importpath/"$(basename "$f")
		var=GoCover_${i}
		go tool cover -mode=set -var ${var} -o "$SRC/$cover" "$f"
		if grep -q '^package main$' "$SRC/$cover"; then
			if grep -q '^func main() {' "$SRC/$cover"; then
				sed -i='' 's/^func main() {$/func __main() {/' "$SRC/$cover"
				mainpkg="$importpath"
			fi
		else
			# since we're not in the main package anymore, we will have to import from the main package
			# the current counter that's specific to the current file
			import+=( "_cover${i} \"$importpath\"" )
			var=_cover${i}.GoCover_${i}
		fi
		init+=( "coverRegisterFile(\"$cover\", ${var}.Count[:], ${var}.Pos[:], ${var}.NumStmt[:])" )
		i=$((i+1))
	done

	if test "$mainpkg" == ""; then
		>&2 echo "main function not found"
		exit 1
	fi

	cat <<-EOF | gofmt > "$SRC/$mainpkg/test_main.go"
		package main

		import (
		  "encoding/gob"
		  "log"
		  "net"
		  "testing"

		$(printf '  %s\n' "${import[@]}")
		)

		var (
		  coverCounters = make(map[string][]uint32)
		  coverBlocks = make(map[string][]testing.CoverBlock)
		)

		func init() {
		$(printf '  %s\n' "${init[@]}")
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

		func main() {
		  __main()

		  c, err := net.Dial("unix", "/tmp/coverify.sock")
		  if err != nil {
		    log.Fatal(err)
		  }
		  enc := gob.NewEncoder(c)
		  if err := enc.Encode(testing.Cover{
		    Mode: "${mode}",
		    Counters: coverCounters,
		    Blocks: coverBlocks,
		  }); err != nil {
		    log.Fatal(err)
		  }
		}
	EOF
	export GOPATH="$WORK"
	go build -o $SRC/$mainpkg/main.test $mainpkg
	echo "$SRC/$mainpkg/main.test"
}

coverify_tests() {
	[ -z "$COVERIFY_BINARY" ] && &>2 echo 'Please set COVERIFY_BINARY environment variable (usually to the output of coverify.sh main)' && exit 1
	o="$(go test -c -x -work $TESTFLAGS $*)"
	eval $(echo "$o" | head -n 1 | grep -m 1 '^WORK=') # set WORK
	compile=$(echo "$o" | grep -m 1 '/compile -o \./main\.a')
	link=$(echo "$o" | grep -m 1 '/link -o ')
	(
		cd "$WORK"
		sed -i='' "s/\(testing.RegisterCover(\)/testing.RegisterCover(getCover());\/\*\1/" _testmain.go
		sed -i='' "s/\(m := testing.MainStart(\)/\*\/\1/" _testmain.go
		cat <<-EOF >> _testmain.go
			func getCover() <-chan testing.Cover {
			  ch := make(chan testing.Cover)
			  go func() {
			    l, err := net.Listen("unix", "/tmp/coverify.sock")
			    if err != nil {
			      log.Println(err)
			    }
			    defer l.Close()
			    for {
			      conn, err := l.Accept()
			      if err != nil {
      			        log.Fatal(err)				      
			      }
			      go func(c net.Conn) {
			        dec := gob.NewDecoder(c)
				var cov testing.Cover
				if err := dec.Decode(&cov); err != nil {
				  log.Fatal(err)
				}
				ch <- cov
			        c.Close()
		      	      }(conn)
			    }
			  }()
			  return ch
			}
		EOF
		$compile
		$link
	)
}

case "$1" in
	main)
		coverify_main ;;
	tests)
		coverify_tests ;;
	*)
		cat <<-EOF
		Usage: coverify.sh main       to produce a coverified binary to be tested. Returns a COVERIFY_BINARY
		       coverify.sh tests      to produce a coverprofile from tests that shell out to the coverified binary (available at COVERIFY_BINARY)
		EOF
esac

