package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	// Test: Create many groups first, then create datasets
	// This triggers multiple B-tree rebuilds and may expose SNOD signature errors
	fmt.Println("=== Testing SNOD signature error scenario ===")
	
	for _, groupCount := range []int{20, 50, 100, 150, 200} {
		filename := fmt.Sprintf("test_snod_groups_%d.h5", groupCount)
		fmt.Printf("\nCreating %s with %d groups then datasets...\n", filename, groupCount)
		
		f, err := hdf5.CreateForWrite(filename, hdf5.CreateTruncate)
		if err != nil {
			fmt.Printf("Failed to create file: %v\n", err)
			return
		}
		
		// Create groups first
		for i := 0; i < groupCount; i++ {
			groupPath := fmt.Sprintf("/group_%04d", i)
			if _, err := f.CreateGroup(groupPath); err != nil {
				fmt.Printf("Failed to create group %s: %v\n", groupPath, err)
				f.Close()
				return
			}
		}
		fmt.Printf("Created %d groups\n", groupCount)
		
		// Now create datasets - this triggers B-tree rebuild
		for i := 0; i < 50; i++ {
			datasetPath := fmt.Sprintf("/dataset_%04d", i)
			if _, err := f.CreateDataset(datasetPath, hdf5.Float64, []uint64{2}); err != nil {
				fmt.Printf("FAILED to create dataset %s: %v\n", datasetPath, err)
				f.Close()
				return
			}
			if i%10 == 0 {
				fmt.Printf("Created dataset %04d\n", i)
			}
		}
		
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to close file: %v\n", err)
			return
		}
		fmt.Printf("Created %s successfully\n", filename)
	}
	
	fmt.Println("\nAll tests completed!")
}
