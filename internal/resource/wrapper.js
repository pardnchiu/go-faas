const fs = require('fs');

const input = fs.readFileSync(0, 'utf-8');
const event = JSON.parse(input);

global.event = event;
global.input = event;

const scriptPath = process.argv[2];
require(scriptPath);