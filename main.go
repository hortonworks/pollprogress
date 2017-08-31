package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sethgrid/multibar"

	"gopkg.in/yaml.v2"
)

func poll(cmd string) (int, int) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		fmt.Printf("[OUTPUT] %s\n", out)
		log.Fatal(err)
	}

	parts := strings.Split(string(out), "/")
	if len(parts) != 2 {
		log.Fatalf("Command should have returned <actual>/<total> instead it was: %s", out)
	}

	act, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		log.Fatal(err)
	}

	sum, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		log.Fatal(err)
	}
	return act, sum
}

func main() {
	log.Println("poll progress ...")
	if len(os.Args) < 2 {
		log.Fatal("yaml file is required")
	}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("Cloudn't read file: %v", err)
	}
	var obj map[string]string
	err = yaml.Unmarshal([]byte(data), &obj)

	progressBars, _ := multibar.New()
	progressBars.Println("Azure blob copy progress:")

	wg := &sync.WaitGroup{}
	wg.Add(len(obj))

	totals := map[string]int{}
	for task, cmd := range obj {
		go func(task, cmd string) {
			_, sum := poll(cmd)
			totals[task] = sum
			wg.Done()
		}(task, cmd)
	}
	wg.Wait()

	wg.Add(len(obj))
	for task, cmd := range obj {
		p := progressBars.MakeBar(totals[task], fmt.Sprintf("%-30s", task))
		go func(task, cmd string, progressFn multibar.ProgressFunc) {

			act, sum := poll(cmd)
			progressFn(act)
			for act < sum {
				act, _ = poll(cmd)
				progressFn(act)
			}

			wg.Done()
		}(task, cmd, p)
	}

	go progressBars.Listen()
	//time.Sleep(time.Second * 3)

	for _, b := range progressBars.Bars {
		b.Update(0)
	}

	wg.Wait()
	time.Sleep(time.Second * 1)

	fmt.Println("DONE")
}
