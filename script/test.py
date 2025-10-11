import sys
import json

def handler(event):
  name = event.get('name', 'World')
  
  return {
    'message': f'Hello {name}!',
    'language': 'Python'
  }

if __name__ == '__main__':
  input_data = sys.argv[1] if len(sys.argv) > 1 else '{}'
  event = json.loads(input_data)
  
  result = handler(event)
  print(json.dumps(result, ensure_ascii=False))