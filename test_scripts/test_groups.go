package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	// Create test file with many groups
	fmt.Println("Creating test_groups.h5 with multiple groups...")
	f, err := hdf5.CreateForWrite("test_groups.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f.Close()

	// Create multiple groups
	for i := 0; i < 100; i++ {
		groupPath := fmt.Sprintf("/group_%03d", i)
		group, err := f.CreateGroup(groupPath)
		if err != nil {
			fmt.Printf("Failed to create group %s: %v\n", groupPath, err)
			return
		}
		// Write a simple attribute to each group
		err = group.WriteAttribute("index", int32(i))
		if err != nil {
			fmt.Printf("Failed to write attribute to group %s: %v\n", groupPath, err)
			return
		}
	}

	fmt.Println("Created 100 groups successfully")

	// Create nested groups
	_, err = f.CreateGroup("/nested")
	if err != nil {
		fmt.Printf("Failed to create group /nested: %v\n", err)
		return
	}
	for i := 0; i < 50; i++ {
		parentPath := fmt.Sprintf("/nested/level1_%02d", i)
		childPath := fmt.Sprintf("%s/level2", parentPath)
		_, err := f.CreateGroup(parentPath)
		if err != nil {
			fmt.Printf("Failed to create group %s: %v\n", parentPath, err)
			return
		}
		group, err := f.CreateGroup(childPath)
		if err != nil {
			fmt.Printf("Failed to create group %s: %v\n", childPath, err)
			return
		}
		err = group.WriteAttribute("value", float64(i*2))
		if err != nil {
			fmt.Printf("Failed to write attribute to group %s: %v\n", childPath, err)
			return
		}
	}

	fmt.Println("Created 50 nested groups successfully")
	fmt.Println("\nFile test_groups.h5 created successfully!")
}
