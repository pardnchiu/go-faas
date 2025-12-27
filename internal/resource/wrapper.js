#!/usr/bin/env node

const fs = require('fs');

// Read script path from command line
const scriptPath = process.argv[2];

if (!scriptPath) {
  console.error('Usage: node wrapper.js <script.js>');
  process.exit(1);
}

// Read stdin (input data)
let inputData = '';
process.stdin.setEncoding('utf8');

process.stdin.on('data', (chunk) => {
  inputData += chunk;
});

process.stdin.on('end', () => {
  try {
    // Parse input JSON
    const event = inputData ? JSON.parse(inputData) : {};
    const input = event;

    // Make event and input available globally
    global.event = event;
    global.input = input;

    // Execute user script wrapped so top-level `return` works
    try {
      const vm = require('vm');
      const code = fs.readFileSync(scriptPath, 'utf8');
      const wrapped = `(async function(){\n${code}\n})()`;
      const scriptObj = new vm.Script(wrapped, { filename: scriptPath });
      const context = vm.createContext(global);
      Promise.resolve(scriptObj.runInContext(context)).then((res) => {
        if (typeof res !== 'undefined') {
          console.log(JSON.stringify(res));
        }
      }).catch((err) => {
        console.error('Error:', err && err.message ? err.message : String(err));
        process.exit(1);
      });
    } catch (e) {
      console.error('Error:', e && e.message ? e.message : String(e));
      process.exit(1);
    }
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
});
