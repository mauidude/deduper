package main

import "fmt"

func main() {
	m := NewMinHasher(2, 1)

	m.Add("this is a string that we are comparing the minhashes of. hope it works!")
	m.Add("this is a string that we are comparing the minhashes of. i really hope it works!")

	id := m.FindSimilar("this is a string that we are comparing the minhashes of. hope it works!")

	fmt.Println("duplicate: ", id)
}
