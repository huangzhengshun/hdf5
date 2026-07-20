import h5py
import os

# Test reading HDF5 file with many groups
print("Test: Reading go-hdf5 created file with many groups")
try:
    with h5py.File('test_groups.h5', 'r') as f:
        print(f"  File opened successfully")
        
        # List all keys in root
        root_keys = list(f.keys())
        print(f"  Root group keys: {len(root_keys)}")
        
        # Count all groups
        group_count = [0]
        def count_groups(name, obj):
            if isinstance(obj, h5py.Group):
                group_count[0] += 1
        
        f.visititems(count_groups)
        print(f"  Total groups: {group_count[0]}")
        
        # Check a few groups
        for i in [0, 49, 99]:
            group_path = f"/group_{i:03d}"
            if group_path in f:
                g = f[group_path]
                if 'index' in g.attrs:
                    idx = g.attrs['index']
                    print(f"  Group {group_path} attribute 'index' = {idx}")
                else:
                    print(f"  Group {group_path} has no 'index' attribute")
        
        # Check nested groups
        if '/nested' in f:
            nested_keys = list(f['/nested'].keys())
            print(f"  Nested group keys: {len(nested_keys)}")
            
            for i in [0, 24, 49]:
                group_path = f"/nested/level1_{i:02d}/level2"
                if group_path in f:
                    g = f[group_path]
                    if 'value' in g.attrs:
                        val = g.attrs['value']
                        print(f"  Group {group_path} attribute 'value' = {val}")
        
        print("  PASSED")
except Exception as e:
    print(f"  FAILED: {e}")

# Cleanup
if os.path.exists('test_groups.h5'):
    os.remove('test_groups.h5')

print("\nTest completed!")
