package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	fmt.Println("Creating test_btree_split.h5 with many groups to trigger B-tree splits...")
	f, err := hdf5.CreateForWrite("test_btree_split.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f.Close()

	// Create many groups to trigger B-tree splits
	// B-tree node typically holds ~400-500 entries depending on key length
	for i := 0; i < 500; i++ {
		groupPath := fmt.Sprintf("/dataset_%04d", i)
		_, err := f.CreateGroup(groupPath)
		if err != nil {
			fmt.Printf("Failed to create group %s: %v\n", groupPath, err)
			return
		}
		if i%100 == 0 {
			fmt.Printf("Created %d groups...\n", i+1)
		}
	}

	fmt.Println("\nCreated 500 groups successfully")
	fmt.Println("File test_btree_split.h5 created successfully!")
}
