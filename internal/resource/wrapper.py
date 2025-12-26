#!/usr/bin/env python3

import sys
import json

if len(sys.argv) < 2:
    print('Usage: python wrapper.py <script.py>', file=sys.stderr)
    sys.exit(1)

script_path = sys.argv[1]

# Read stdin (input data)
input_data = sys.stdin.read()

try:
    # Parse input JSON
    event = json.loads(input_data) if input_data.strip() else {}
    input_var = event

    # Make event and input available globally
    globals()['event'] = event
    globals()['input'] = input_var

    # Execute user script
    with open(script_path, 'r') as f:
        code = f.read()
        exec(code, globals())

except Exception as e:
    print(f'Error: {str(e)}', file=sys.stderr)
    sys.exit(1)
