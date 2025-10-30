package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync_files/internal/logs"
	"sync_files/internal/stream"
	"syscall"
)

// Проверка наличия диектории
func CheckFile(fName string) error {
	_, err := os.Stat(fName)
	if err != nil {
		return err
	}
	return nil
}

// Проверка введённого числа на корректность
func CheckNum() (int, error) {
	strNum := os.Args[1]
	num, err := strconv.Atoi(strNum)
	if err != nil {
		return 0, err
	}
	if num <= 0 {
		return 0, fmt.Errorf("First argument is not correct")
	}
	return num, nil
}

// Проверка путей к директориям и возвращение числа из аргументак командной строки
func CheckArgs() int {
	if len(os.Args) < 4 {
		log.Fatal("Not enough arguments")
	}

	num, err := CheckNum()
	if err != nil {
		log.Fatal(err)
	}

	for i := 2; i < len(os.Args); i++ {
		if err := CheckFile(os.Args[i]); err != nil {
			log.Fatal("Directory: {", os.Args[i],
				"} Error: ", err)
		}
	}
	return num
}

func main() {
	timeMult := CheckArgs()

	logs.LogsInit()

	//Создание буфера, хранящего информацию о файлах из заданных директорий
	buf, err := stream.SyncInfo(os.Args[2:])
	if err != nil {
		log.Fatal(err)
	}

	sigShut, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup

	//Запуск горутин, синхронизирующих директории с буфером
	for _, file := range os.Args[2:] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf.RunSync(file, timeMult, sigShut)
			cancel()
		}()
	}

	fmt.Println("Synchronisation is started")

	//Graceful shutdown
	<-sigShut.Done()

	fmt.Println("Waiting for all goroutine shut...")

	wg.Wait()

	fmt.Println("Sinchronisation is over")
}
