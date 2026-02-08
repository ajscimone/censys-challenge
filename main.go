package main

import "log"

func Add(x, y int) (int, error) {
	return x + y, nil
}

func main() {
	r, err := Add(42, 42)
	if err != nil {
		log.Println("this is impossible")
	}
	log.Fatal(r)
}
