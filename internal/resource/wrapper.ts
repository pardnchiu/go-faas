import * as fs from 'fs';

const input = fs.readFileSync(0, 'utf-8');
const event = JSON.parse(input);

(global as any).event = event;
(global as any).input = event;

const scriptPath = process.argv[2];
require(scriptPath);