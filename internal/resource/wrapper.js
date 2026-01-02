#!/usr/bin/env node

const vm = require('vm');

// Read stdin (JSON payload with code and input)
let inputData = '';
process.stdin.setEncoding('utf8');

process.stdin.on('data', (chunk) => {
  inputData += chunk;
});

process.stdin.on('end', () => {
  try {
    // Parse payload JSON
    const payload = inputData ? JSON.parse(inputData) : {};
    const code = payload.code || '';
    const inputStr = payload.input || '';

    // Parse input JSON
    const event = inputStr ? JSON.parse(inputStr) : {};
    const input = event;

    // Make event and input available globally
    global.event = event;
    global.input = input;

    // Execute user script wrapped so top-level `return` works
    try {
      const wrapped = `(async function(){\n${code}\n})()`;
      const scriptObj = new vm.Script(wrapped, { filename: 'user-code.js' });
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
