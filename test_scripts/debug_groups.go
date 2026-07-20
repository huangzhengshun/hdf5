package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	fmt.Println("Creating debug_groups.h5 with 260 groups...")
	f, err := hdf5.CreateForWrite("debug_groups.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f.Close()

	// Create groups 240-260 to focus on the problematic range
	for i := 240; i < 260; i++ {
		groupPath := fmt.Sprintf("/dataset_%04d", i)
		_, err := f.CreateGroup(groupPath)
		if err != nil {
			fmt.Printf("Failed to create group %s: %v\n", groupPath, err)
			return
		}
		if i%5 == 0 {
			fmt.Printf("Created group %04d\n", i)
		}
	}

	fmt.Println("\nCreated 20 groups successfully")
	fmt.Println("File debug_groups.h5 created successfully!")
}
