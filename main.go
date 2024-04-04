package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"errors"

	"gopkg.in/yaml.v2"
)

const ERROR_LIMIT = 5

func poll(cmd string) (int, int, error) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("ERROR: output: %s, err: %s", out, err.Error()))
	}

	if strings.Contains(string(out), "ErrorCode:PendingCopyOperation") {
		return 0, 100, nil
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
	errChan := make(chan error, len(tasks))

	log.Println("Getting blob sizes..")
	for storageAccount, cmd := range tasks {
		go func(storageAccount, cmd string) {
			defer wg.Done()
			sum, numErr := 0, 0
			for sum == 0 {
				_, sum, err = poll(cmd)
				if err != nil {
					numErr = numErr + 1
					log.Printf("Error finding the total size of the vhd in Storage Account: %s, error: %s", storageAccount, err.Error())
					if numErr == ERROR_LIMIT {
						errChan <- fmt.Errorf("error limit reached for total size of the vhd in Storage Account %s, exiting with err %w", storageAccount, err)
						delete(tasks, storageAccount) // remove the task for the failed storage account
						return
					}
					time.Sleep(time.Second * 1)
				}
			}
			if sum < 1024 {
				log.Println("Azure didn't return exact numbers, but reported that the blob is being copied... (fingers crossed)")
			} else {
				log.Printf("Blob size of vhd in Storage Account of %s is: %f GB", storageAccount, float64(sum)/math.Pow(1024, 3))
			}
		}(storageAccount, cmd)
	}
	wg.Wait()

	wg.Add(len(tasks))
	for storageAccount, cmd := range tasks {
		go func(storageAccount, cmd string) {
			defer wg.Done()
			act, sum, numErr := 0, 0, 0
			var err error
			for act == 0 || sum == 0 || act < sum {
				act, sum, err = poll(cmd)
				if err != nil {
					numErr = numErr + 1
					log.Printf("Failed to check the copy status to Storage Account %s [%d/%d] err: %s", storageAccount, numErr, ERROR_LIMIT, err.Error())
					if numErr == ERROR_LIMIT {
						errChan <- fmt.Errorf("error limit reached for the copy status to Storage Account %s, exiting with err %w", storageAccount, err)
						return
					}
				} else {
					if sum < 1024 {
						log.Println("Azure didn't return exact numbers, but reported that the blob is still being copied... (fingers crossed)")
					} else {
						log.Printf("Copy status to Storage Account of %s is: (%d/%d) %.2f%% ", storageAccount, act, sum, (float64(act)/float64(sum))*100)
					}
				}
				time.Sleep(time.Second * 10)
			}
		}(storageAccount, cmd)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) != 0 {
		for err := range errChan {
			log.Printf("%s", err.Error())
		}
		panic("Failed to poll progress of some copy operations")
	}

	log.Println("DONE")
}
