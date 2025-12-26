#!/usr/bin/env tsx

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

    // Execute user script
    await import(scriptPath);
  } catch (error: any) {
    console.error('Error:', error.message);
    process.exit(1);
  }
});
