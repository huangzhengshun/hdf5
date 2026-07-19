import h5py
import numpy as np

print("Creating h5py-written test file...")

with h5py.File('h5py_written.h5', 'w') as f:
    f.create_dataset('int32_data', data=np.arange(100, dtype=np.int32))
    f.create_dataset('float64_data', data=np.linspace(0, 1, 50, dtype=np.float64))
    f.create_dataset('compressed_data', data=np.arange(1000, dtype=np.int32), compression='gzip', compression_opts=6)
    f.create_dataset('shuffled_compressed', data=np.arange(1000, dtype=np.int32), compression='gzip', shuffle=True)
    
    grp = f.create_group('test_group')
    grp.create_dataset('nested_data', data=np.arange(10, dtype=np.uint8))

print("File created: h5py_written.h5")
