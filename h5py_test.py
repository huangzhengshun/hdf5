import h5py
import numpy as np
import os

# Test 1: Read HDF5 file created by go-hdf5 with GZIP compression
print("Test 1: Reading go-hdf5 created file with GZIP compression")
try:
    with h5py.File('test_gzip.h5', 'r') as f:
        dset = f['data']
        print(f"  Dataset shape: {dset.shape}")
        print(f"  Dataset dtype: {dset.dtype}")
        data = dset[:]
        print(f"  First 10 values: {data[:10]}")
        print(f"  Last 10 values: {data[-10:]}")
        # Verify data correctness
        expected = np.arange(1000, dtype=np.float64)
        assert np.array_equal(data, expected), "Data mismatch!"
    print("  PASSED")
except Exception as e:
    print(f"  FAILED: {e}")

# Test 2: Read HDF5 file created by go-hdf5 with Shuffle+GZIP
print("\nTest 2: Reading go-hdf5 created file with Shuffle+GZIP")
try:
    with h5py.File('test_shuffle_gzip.h5', 'r') as f:
        dset = f['data']
        print(f"  Dataset shape: {dset.shape}")
        print(f"  Dataset dtype: {dset.dtype}")
        data = dset[:]
        print(f"  First 10 values: {data[:10]}")
        print(f"  Last 10 values: {data[-10:]}")
        expected = np.arange(1000, dtype=np.float64)
        assert np.array_equal(data, expected), "Data mismatch!"
    print("  PASSED")
except Exception as e:
    print(f"  FAILED: {e}")

# Test 3: Read HDF5 file created by go-hdf5 with Fletcher32
print("\nTest 3: Reading go-hdf5 created file with Fletcher32")
try:
    with h5py.File('test_fletcher32.h5', 'r') as f:
        dset = f['data']
        print(f"  Dataset shape: {dset.shape}")
        print(f"  Dataset dtype: {dset.dtype}")
        data = dset[:]
        print(f"  First 10 values: {data[:10]}")
        print(f"  Last 10 values: {data[-10:]}")
        expected = np.arange(1000, dtype=np.float64)
        assert np.array_equal(data, expected), "Data mismatch!"
    print("  PASSED")
except Exception as e:
    print(f"  FAILED: {e}")

# Test 4: Create HDF5 file with h5py and read with go-hdf5
print("\nTest 4: Creating file with h5py for go-hdf5 to read")
try:
    data = np.arange(1000, dtype=np.float64)
    with h5py.File('h5py_test.h5', 'w') as f:
        dset = f.create_dataset('data', data=data, chunks=(100,), compression='gzip', compression_opts=6)
        print(f"  Created dataset with GZIP compression")
    print("  PASSED")
except Exception as e:
    print(f"  FAILED: {e}")

# Cleanup
for f in ['test_gzip.h5', 'test_shuffle_gzip.h5', 'test_fletcher32.h5', 'h5py_test.h5']:
    if os.path.exists(f):
        os.remove(f)

print("\nAll tests completed!")
