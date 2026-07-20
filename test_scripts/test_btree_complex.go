package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	// Test 1: Create groups first, then datasets (this triggers different B-tree rebuild scenarios)
	fmt.Println("=== Test 1: Create groups first, then datasets ===")
	f, err := hdf5.CreateForWrite("test_btree_groups_first.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	
	// Create 20 groups
	for i := 0; i < 20; i++ {
		groupPath := fmt.Sprintf("/group_%03d", i)
		if _, err := f.CreateGroup(groupPath); err != nil {
			fmt.Printf("Failed to create group %s: %v\n", groupPath, err)
			f.Close()
			return
		}
	}
	fmt.Println("Created 20 groups")
	
	// Create 200 datasets (total 220 entries, which should trigger B-tree splits)
	for i := 0; i < 200; i++ {
		datasetPath := fmt.Sprintf("/dataset_%04d", i)
		if _, err := f.CreateDataset(datasetPath, hdf5.Float64, []uint64{2}); err != nil {
			fmt.Printf("Failed to create dataset %s: %v\n", datasetPath, err)
			f.Close()
			return
		}
		if i%50 == 0 {
			fmt.Printf("Created dataset %04d\n", i)
		}
	}
	
	if err := f.Close(); err != nil {
		fmt.Printf("Failed to close file: %v\n", err)
		return
	}
	fmt.Println("Created test_btree_groups_first.h5 successfully")
	
	// Test 2: Create datasets in a sub-group
	fmt.Println("\n=== Test 2: Create datasets in sub-group ===")
	f2, err := hdf5.CreateForWrite("test_btree_subgroup.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	
	// Create a sub-group
	if _, err := f2.CreateGroup("/subgroup"); err != nil {
		fmt.Printf("Failed to create subgroup: %v\n", err)
		f2.Close()
		return
	}
	
	// Create 200 datasets in the sub-group
	for i := 0; i < 200; i++ {
		datasetPath := fmt.Sprintf("/subgroup/dataset_%04d", i)
		if _, err := f2.CreateDataset(datasetPath, hdf5.Float64, []uint64{2}); err != nil {
			fmt.Printf("Failed to create dataset %s: %v\n", datasetPath, err)
			f2.Close()
			return
		}
		if i%50 == 0 {
			fmt.Printf("Created dataset %04d in subgroup\n", i)
		}
	}
	
	if err := f2.Close(); err != nil {
		fmt.Printf("Failed to close file: %v\n", err)
		return
	}
	fmt.Println("Created test_btree_subgroup.h5 successfully")
	
	// Test 3: Mixed scenario - create groups and datasets interleaved
	fmt.Println("\n=== Test 3: Mixed groups and datasets ===")
	f3, err := hdf5.CreateForWrite("test_btree_mixed.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	
	// Create groups and datasets interleaved
	for i := 0; i < 100; i++ {
		if i%3 == 0 {
			groupPath := fmt.Sprintf("/group_%03d", i)
			if _, err := f3.CreateGroup(groupPath); err != nil {
				fmt.Printf("Failed to create group %s: %v\n", groupPath, err)
				f3.Close()
				return
			}
		} else {
			datasetPath := fmt.Sprintf("/dataset_%04d", i)
			if _, err := f3.CreateDataset(datasetPath, hdf5.Float64, []uint64{2}); err != nil {
				fmt.Printf("Failed to create dataset %s: %v\n", datasetPath, err)
				f3.Close()
				return
			}
		}
	}
	
	if err := f3.Close(); err != nil {
		fmt.Printf("Failed to close file: %v\n", err)
		return
	}
	fmt.Println("Created test_btree_mixed.h5 successfully")
	
	fmt.Println("\nAll complex tests completed!")
}
