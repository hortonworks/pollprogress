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
	"math"

	"github.com/sethgrid/multibar"

	"errors"
	"gopkg.in/yaml.v2"
)

func poll(cmd string) (int, int, error) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("ERROR: output: %s, err: %s", out, err.Error()))
	}

	parts := strings.Split(string(out), "/")
	if len(parts) != 2 {
		return 0, 0, errors.New(fmt.Sprintf("Command should have returned <actual>/<total> instead it was: %s", out))
	}

	act, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("Cannot convert actual copy progress to int: %s", parts[0]))
	}

	secondPart := strings.Split(parts[1],"\n")

	sum, err := strconv.Atoi(strings.TrimSpace(secondPart[0]))
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("Cannot convert total blob size to int: %s", secondPart[0]))
	}

	status := strings.TrimSpace(secondPart[1])
	switch status {
	    case
	        "pending",
					"success":
	        return act, sum, nil
	    }
	return 0, 0, errors.New(fmt.Sprintf("Copy operation status is invalid: %s", status))
}

func main() {
	log.Println("poll progress ...")
	if len(os.Args) < 2 {
		log.Fatal("yaml file is required")
	}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("Couldn't read file: %v", err)
	}
	var obj map[string]string
	err = yaml.Unmarshal([]byte(data), &obj)

	progressBars, _ := multibar.New()
	progressBars.Println("Azure blob copy progress:")

	wg := &sync.WaitGroup{}
	wg.Add(len(obj))

	totals := map[string]int{}
	log.Println("Getting blob sizes..")
	for task, cmd := range obj {
		go func(task, cmd string) {
			defer wg.Done()
			_, sum, err := poll(cmd)
			for err != nil {
				time.Sleep(time.Second * 1)
				log.Printf("Error finding the total size of the vhd in Storage Account: %s, error: %s", task, err.Error())
				_, sum, err = poll(cmd)
			}
			log.Printf("Blob size of vhd in Storage Account of %s is: %f GB", task, float64(sum) / math.Pow(1024,3))
			totals[task] = sum
		}(task, cmd)
	}
	wg.Wait()

	wg.Add(len(obj))
	for task, cmd := range obj {
		p := progressBars.MakeBar(totals[task], fmt.Sprintf("%-30s", task))
		go func(cmd string, progressFn multibar.ProgressFunc) {
			defer wg.Done()
			act, sum, err := poll(cmd)
			if err == nil {
				progressFn(act)
			}
			for act < sum || err != nil {
				time.Sleep(time.Second * 1)
				act, _, err = poll(cmd)
				if err == nil {
					progressFn(act)
				}
			}
			progressFn(act)

		}(cmd, p)
	}

	go progressBars.Listen()

	for _, b := range progressBars.Bars {
		b.Update(0)
	}

	wg.Wait()

	fmt.Println("DONE")
}
