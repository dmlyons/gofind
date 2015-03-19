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

var showErrors bool

func main() {
	var where, what string
	var numWorkers int
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.BoolVar(&showErrors, `e`, false, `Show errors along the way`)
	flag.IntVar(&numWorkers, `w`, 100, `Number of workers to use`)
	flag.Parse()
	where = flag.Arg(0)
	what = flag.Arg(1)
	fmt.Printf("Searching %s for %s\n", where, what)

	jobs := make(chan search, 100000)
	results := make(chan string, 10)

	// launch the workers
	for i := 1; i <= numWorkers; i++ {
		go findInDirWorker(i, jobs, results)
	}
	go showResults(results)

	wg.Add(1)
	jobs <- search{what, where}
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
			if showErrors {
				fmt.Printf("DirErr: %v\n", err)
			}
			wg.Done()
			continue
		}
		for _, f := range d {
			path, err := filepath.Abs(j.where)
			if err != nil {
				if showErrors {
					fmt.Printf("DirErr: %v\n", err)
				}
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
