#!/usr/bin/env tsx

import * as fs from 'fs';
import * as vm from 'vm';

// Read script path from command line
const scriptPath = process.argv[2];

if (!scriptPath) {
  console.error('Usage: tsx wrapper.ts <script.ts>');
  process.exit(1);
}

// Read stdin (input data)
let inputData = '';
process.stdin.setEncoding('utf8');

process.stdin.on('data', (chunk) => {
  inputData += chunk;
});

process.stdin.on('end', async () => {
  try {
    // Parse input JSON
    const event = inputData ? JSON.parse(inputData) : {};
    const input = event;

    // Make event and input available globally
    (global as any).event = event;
    (global as any).input = input;

    // Execute user script wrapped so top-level `return` works
    try {
      const code = fs.readFileSync(scriptPath, 'utf8');
      const wrapped = `(async function(){\n${code}\n})()`;
      const scriptObj = new vm.Script(wrapped, { filename: scriptPath });
      const context = vm.createContext(global as any);
      const res = await Promise.resolve(scriptObj.runInContext(context));
      if (typeof res !== 'undefined') {
        (global as any).__return__ = res;
      }
    } catch (e) {
      console.error('Error:', e && (e as any).message ? (e as any).message : String(e));
      process.exit(1);
    }
  } catch (error: any) {
    console.error('Error:', error.message);
    process.exit(1);
  }
  // If script set (global as any).result or (global as any).__return__, print it as JSON
  try {
    const g: any = globalThis as any;
    const res = g.result ?? g.__return__;
    if (typeof res !== 'undefined') {
      // print as single JSON line
      // eslint-disable-next-line no-console
      console.log(JSON.stringify(res));
    }
  } catch (e) {
    // ignore
  }
});
