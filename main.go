package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type search struct {
	what, where string
}

var wg sync.WaitGroup

func main() {
	var where, what string

	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	where = flag.Arg(0)
	what = flag.Arg(1)
	fmt.Printf("Searching %s for %s\n", where, what)

	jobs := make(chan search, 100000)
	results := make(chan string, 100000)

	// launch the workers
	for i := 1; i <= 10; i++ {
		go findInDirWorker(i, jobs, results)
	}

	wg.Add(1)
	jobs <- search{what, where}
	go showResults(results)
	wg.Wait()
	close(jobs)
	close(results)
}

func showResults(c chan string) {
	n := 0
	for found := range c {
		n++
		fmt.Printf("%d. %s\n", n, found)
	}
}

func findInDirWorker(w int, jobs chan search, result chan<- string) {
	for j := range jobs {
		d, err := ioutil.ReadDir(j.where)
		if err != nil {
			//fmt.Printf("DirErr: %v\n", err)
			wg.Done()
			continue
		}
		for _, f := range d {
			path, err := filepath.Abs(j.where)
			if err != nil {
				continue
			}
			full := fmt.Sprintf("%s%v%s", path, string(os.PathSeparator), f.Name())
			if strings.Contains(f.Name(), j.what) {
				res := fmt.Sprintf("%s (%d)", full, w)
				result <- res
			}
			if f.IsDir() {
				wg.Add(1)
				jobs <- search{j.what, full}
			}
		}
		wg.Done()
	}
}
