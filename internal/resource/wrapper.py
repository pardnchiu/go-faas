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

    # Execute user script wrapped in a function so top-level `return` works
    with open(script_path, 'r') as f:
        code = f.read()

    func_code = 'def __user_main__():\n'
    for line in code.splitlines():
        func_code += '    ' + line + '\n'

    exec(func_code, globals())
    result = globals()['__user_main__']()

except Exception as e:
    print(f'Error: {str(e)}', file=sys.stderr)
    sys.exit(1)

# If the script returned a value or set a `result`/`__return__` global, print it as JSON
try:
    if 'result' in globals():
        print(json.dumps(globals()['result']))
    elif '__return__' in globals():
        print(json.dumps(globals()['__return__']))
    else:
        # prefer returned value if available
        try:
            print(json.dumps(result))
        except Exception:
            pass
except Exception:
    # ignore serialization errors; leave any prints as-is
    pass
