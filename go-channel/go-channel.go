package main

import "fmt"

func main() {
	// dataChan := make(chan string)

	// go func() {
	// 	dataChan <- "Hello, Channel!"
	// }()

	// c := <-dataChan
	// fmt.Println(c)

	dataChan := make(chan string, 1) // buffer size = 1

	dataChan <- "Hello, Channel!"

	c := <-dataChan
	fmt.Println(c)
}
