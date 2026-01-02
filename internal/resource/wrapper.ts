#!/usr/bin/env tsx

import { createRequire } from 'module';
import * as vm from 'vm';

const require = createRequire(import.meta.url);

// Read stdin (JSON payload with code and input)
let inputData = '';
process.stdin.setEncoding('utf8');

process.stdin.on('data', (chunk) => {
  inputData += chunk;
});

process.stdin.on('end', async () => {
  try {
    // Parse payload JSON
    const payload = inputData ? JSON.parse(inputData) : {};
    const code = payload.code || '';
    const inputStr = payload.input || '';

    // Parse input JSON
    const event = inputStr ? JSON.parse(inputStr) : {};
    const input = event;

    // Make event and input available globally
    (global as any).event = event;
    (global as any).input = input;

    // Execute user script: compile TypeScript then run with vm
    try {
      // Compile TypeScript to JavaScript with esbuild
      const { transformSync } = require('esbuild');
      const result = transformSync(code, {
        loader: 'ts',
        format: 'cjs',
        target: 'node18'
      });

      const jsCode = result.code;

      // Wrap in async IIFE to support top-level return
      const wrapped = `(async function(){\n${jsCode}\n})()`;
      const scriptObj = new vm.Script(wrapped, { filename: 'user-code.ts' });
      const context = vm.createContext(global as any);
      const res = await Promise.resolve(scriptObj.runInContext(context));

      if (typeof res !== 'undefined') {
        (global as any).__return__ = res;
      }
    } catch (e: any) {
      console.error('Error:', e.message || String(e));
      process.exit(1);
    }
  } catch (error: any) {
    console.error('Error:', error.message);
    process.exit(1);
  }

  // Output result
  try {
    const g: any = globalThis as any;
    const res = g.result ?? g.__return__;
    if (typeof res !== 'undefined') {
      console.log(JSON.stringify(res));
    }
  } catch (e) {
    // ignore
  }
});
