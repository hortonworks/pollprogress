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

	secondPart := strings.Split(parts[1], "\n")

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
	var tasks map[string]string
	err = yaml.Unmarshal([]byte(data), &tasks)

	log.Println("Azure blob copy progress tracking started")

	wg := &sync.WaitGroup{}
	wg.Add(len(tasks))

	totals := map[string]int{}
	log.Println("Getting blob sizes..")
	for storageAccount, cmd := range tasks {
		go func(storageAccount, cmd string) {
			defer wg.Done()
			_, sum, err := poll(cmd)
			for err != nil {
				time.Sleep(time.Second * 1)
				log.Printf("Error finding the total size of the vhd in Storage Account: %s, error: %s", storageAccount, err.Error())
				_, sum, err = poll(cmd)
			}
			log.Printf("Blob size of vhd in Storage Account of %s is: %f GB", storageAccount, float64(sum)/math.Pow(1024, 3))
			totals[storageAccount] = sum
		}(storageAccount, cmd)
	}
	wg.Wait()

	wg.Add(len(tasks))
	for storageAccount, cmd := range tasks {
		go func(storageAccount, cmd string) {
			defer wg.Done()
			act, sum, err := poll(cmd)
			if err == nil {
				log.Printf("Copy status to Storage Account of %s is: (%d/%d) %.2f%% ", storageAccount, act, sum, (float64(act)/float64(sum))*100)
			}
			for act < sum || err != nil {
				time.Sleep(time.Second * 10)
				act, sum, err = poll(cmd)
				if err == nil {
					log.Printf("Copy status to Storage Account of %s is: (%d/%d) %.2f%% ", storageAccount, act, sum, (float64(act)/float64(sum))*100)
				}
			}
		}(storageAccount, cmd)
	}

	wg.Wait()

	fmt.Println("DONE")
}
