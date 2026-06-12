package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	resp, err := http.Post(endpoint, "text/plain", strings.NewReader(long))
	if err != nil {
		panic(err)
	}

	fmt.Println("Статус-код", resp.Status)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
