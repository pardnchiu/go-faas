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

    // Execute user script
    require(scriptPath);
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
});
