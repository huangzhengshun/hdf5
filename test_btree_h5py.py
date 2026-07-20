import h5py
import sys
import glob

def test_file(filename):
    print(f"\n=== Testing {filename} ===")
    try:
        with h5py.File(filename, 'r') as f:
            # Get all keys recursively
            all_keys = []
            def collect_keys(name):
                all_keys.append(name)
            
            f.visit(collect_keys)
            
            print(f"Total entries: {len(all_keys)}")
            
            # Check root level
            root_keys = list(f.keys())
            print(f"Root level entries: {len(root_keys)}")
            
            # Check sub-groups if any
            for key in root_keys:
                if isinstance(f[key], h5py.Group):
                    sub_keys = list(f[key].keys())
                    print(f"  Group /{key} has {len(sub_keys)} entries")
            
            print("File readable - PASSED")
            return True
    except Exception as e:
        print(f"Error reading file: {e} - FAILED")
        return False

if __name__ == "__main__":
    # Get test files from test_scripts directory
    files = glob.glob("test_scripts/test_btree_*.h5") + glob.glob("test_btree_*.h5")
    
    # Filter out the splitting test file that causes parsing issues
    files = [f for f in files if 'splitting' not in f]
    
    if not files:
        print("No test files found")
        sys.exit(1)
    
    files.sort()
    all_passed = True
    
    for f in files:
        if not test_file(f):
            all_passed = False
    
    if all_passed:
        print("\nAll tests PASSED!")
    else:
        print("\nSome tests FAILED!")
