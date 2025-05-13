package main

import (
	"fmt"
	"os"
	"context"
	"strconv"
	"log"
	"log/slog"
	"sync_files/internal/stream"
)


//Проверка наличия диектории
func CheckFile(fName string) error {
	_, err := os.Stat(fName)
	if err != nil {
		return err
	}
	return nil
}

//Проверка введённого числа на корректность
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

//Проверка путей к директориям и возвращение числа из аргументак командной строки
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

	logFile, err := os.Create("log.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
        logger := slog.New(slog.NewTextHandler(logFile, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf, err := stream.SyncInfo(os.Args[2:])
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, file := range os.Args[2:] {
		go buf.RunSync(file, timeMult, cancel)
	}


	fmt.Println("Synchronisation is started")
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Synchronisation is over")
			return
		}
	}
}
