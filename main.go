/*
This command is a partial implementation of GNU find used to play around with
channels, concurrency, worker pools, and other fun go stuff.
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// search is a struct representing an individual search
type search struct {
	what, where string
}

var showErrors bool

func main() {
	var where, what string
	var numWorkers int
	wg := &sync.WaitGroup{}

	flag.BoolVar(&showErrors, `e`, false, `Show errors along the way`)
	flag.IntVar(&numWorkers, `w`, 100, `Number of workers to use`)
	flag.Parse()
	where = flag.Arg(0)
	what = flag.Arg(1)
	fmt.Printf("Searching %s for %s\n", where, what)

	// jobs is a collection of searches.  As new directories are encountered,
	// they will be added to the jobs queue
	// 100K is a bit arbitrary, but have to limit it somewhere
	jobs := make(chan search, 100000)
	defer close(jobs)

	results := make(chan string, 10)

	// launch the workers
	for i := 1; i <= numWorkers; i++ {
		go findInDirWorker(i, jobs, results, wg)
	}

	// Separate waitgroup for the results, to make sure we get through all of them
	rwg := &sync.WaitGroup{}
	rwg.Add(1)
	go showResults(results, rwg)

	wg.Add(1)
	jobs <- search{what, where}
	wg.Wait()

	// close the results channel so that showResults can finish
	close(results)
	// wait for showResults to finish
	rwg.Wait()
}

// showResults cycles through the results channel and sends them to stdout
func showResults(c chan string, wg *sync.WaitGroup) {
	n := 0
	for found := range c {
		n++
		fmt.Printf("%d. %s\n", n, found)
	}
	// This is mainly here so that I am sure it got through the results
	fmt.Println("DONE!!")
	wg.Done()
}

// findInDirWorker takes a "worker number", a jobs channel, and a results
// channel, it will pull a job off the jobs channel and push the results of its
// search into the results channel.  If it encounters a directory, it will add
// it to the jobs channel.
func findInDirWorker(w int, jobs chan search, result chan<- string, wg *sync.WaitGroup) {
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
			// is it a match? add to results
			if strings.Contains(f.Name(), j.what) {
				res := fmt.Sprintf("%s (%d)", full, w)
				result <- res
			}
			// if it is a directory, add it as a job
			if f.IsDir() {
				wg.Add(1)
				jobs <- search{j.what, full}
			}
		}
		wg.Done()
	}

}
