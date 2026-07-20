import h5py
import os

print("Test: Reading debug_missing.h5")
try:
    with h5py.File('debug_missing.h5', 'r') as f:
        print(f"  File opened successfully")
        
        all_keys = list(f.keys())
        print(f"  Total keys found: {len(all_keys)}")
        
        # Check groups 240-260
        print("\n  Checking groups 240-260:")
        for i in range(240, 260):
            group_path = f"/dataset_{i:04d}"
            if group_path in f:
                print(f"    {group_path}: OK")
            else:
                print(f"    {group_path}: MISSING")
        
        # Also check the full range
        print("\n  Checking groups 0-20:")
        for i in range(0, 21):
            group_path = f"/dataset_{i:04d}"
            if group_path in f:
                print(f"    {group_path}: OK")
            else:
                print(f"    {group_path}: MISSING")
        
        print("\n  PASSED")
except Exception as e:
    import traceback
    print(f"  FAILED: {e}")
    traceback.print_exc()

# Cleanup
if os.path.exists('debug_missing.h5'):
    os.remove('debug_missing.h5')

print("\nTest completed!")
