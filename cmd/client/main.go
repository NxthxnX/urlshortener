package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
)

const endpoint = "http://localhost:8080"

func main() {
	fmt.Println("Введите длинный URL")

	reader := bufio.NewReader(os.Stdin)

	long, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	long = strings.TrimSpace(long)

	client := resty.New()

	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		SetBody(long).
		Post(endpoint)

	if err != nil {
		panic(err)
	}

	fmt.Println("Статус-код", resp.Status())
	fmt.Println(string(resp.Body()))
}
