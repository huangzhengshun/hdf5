package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	fmt.Println("Creating debug_missing.h5 with groups 240-260...")
	f, err := hdf5.CreateForWrite("debug_missing.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f.Close()

	for i := 0; i < 260; i++ {
		groupPath := fmt.Sprintf("/dataset_%04d", i)
		_, err := f.CreateGroup(groupPath)
		if err != nil {
			fmt.Printf("Failed to create group %s: %v\n", groupPath, err)
			return
		}
		if i >= 240 && i%5 == 0 {
			fmt.Printf("Created group %04d\n", i)
		}
	}

	fmt.Println("\nCreated 260 groups successfully")
	fmt.Println("File debug_missing.h5 created successfully!")
}
