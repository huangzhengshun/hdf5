package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	fmt.Println("Creating test_dataset_link.h5 with 500 datasets...")
	f, err := hdf5.CreateForWrite("test_dataset_link.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f.Close()

	for i := 0; i < 500; i++ {
		datasetPath := fmt.Sprintf("/dataset_%04d", i)
		_, err := f.CreateDataset(datasetPath, hdf5.Float64, []uint64{2})
		if err != nil {
			fmt.Printf("Failed to create dataset %s: %v\n", datasetPath, err)
			return
		}
		if i%100 == 0 {
			fmt.Printf("Created dataset %04d\n", i)
		}
	}

	fmt.Println("\nCreated 500 datasets successfully")
	fmt.Println("File test_dataset_link.h5 created successfully!")
}
