import sys
import json
import importlib.util

# * disable `__pycache__` 
sys.dont_write_bytecode = True

input_data = sys.stdin.read()
event = json.loads(input_data)

import builtins
builtins.event = event
builtins.input = event

script_path = sys.argv[1]
spec = importlib.util.spec_from_file_location("user_script", script_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)