package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/mauidude/deduper/minhash"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	m := minhash.New(100, 2, 2)
	dir := "/Users/shane/Desktop/dups"
	files, _ := ioutil.ReadDir(dir)

	for i, file := range files {
		fmt.Println("Adding", file.Name())
		path := filepath.Join(dir, file.Name())
		bytes, _ := ioutil.ReadFile(path)

		m.Add(path, string(bytes))
		fmt.Printf("%d/%d\n", i, len(files))
	}

	//fmt.Println("duplicate: ", id)
}
