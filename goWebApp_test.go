package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

func TestClient(t *testing.T) {
	resp, err := http.Get("http://localhost:8082/hello")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	var buffer bytes.Buffer
	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		buffer.WriteString(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
