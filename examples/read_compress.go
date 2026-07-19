package main

import (
	"fmt"
	"os"

	"github.com/huangzhengshun/hdf5"
)

func main() {
	testFile := "read_compress_test.h5"
	os.Remove(testFile)

	// Create compressed dataset
	fw, err := hdf5.CreateForWrite(testFile, hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}

	dsWrite, err := fw.CreateDataset("/data", hdf5.Int32, []uint64{10}, hdf5.WithChunkDims([]uint64{5}), hdf5.WithGZIPCompression(6))
	if err != nil {
		fmt.Printf("Failed to create dataset: %v\n", err)
		return
	}

	data := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	err = dsWrite.Write(data)
	if err != nil {
		fmt.Printf("Failed to write data: %v\n", err)
		return
	}

	err = fw.Close()
	if err != nil {
		fmt.Printf("Failed to close file: %v\n", err)
		return
	}

	// Read back the file
	fr, err := hdf5.Open(testFile)
	if err != nil {
		fmt.Printf("Failed to open file for reading: %v\n", err)
		return
	}
	defer fr.Close()

	root := fr.Root()
	children := root.Children()
	var ds *hdf5.Dataset
	for _, child := range children {
		if child.Name() == "data" {
			var ok bool
			ds, ok = child.(*hdf5.Dataset)
			if !ok {
				fmt.Printf("Not a dataset\n")
				return
			}
			break
		}
	}
	if ds == nil {
		fmt.Printf("Dataset not found\n")
		return
	}

	readData, err := ds.Read()
	if err != nil {
		fmt.Printf("Failed to read data: %v\n", err)
		return
	}

	fmt.Printf("Original data: %v\n", data)
	fmt.Printf("Read data: %v\n", readData)

	match := true
	for i, v := range data {
		if int32(readData[i]) != v {
			match = false
			break
		}
	}

	if match {
		fmt.Println("✓ Data matches! Project can read its own compressed files.")
	} else {
		fmt.Println("✗ Data mismatch!")
	}
}
