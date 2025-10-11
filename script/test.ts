interface Event {
  name?: string;
  [key: string]: any;
}

interface Result {
  message: string;
  info: string;
  language: string;
}

function handler(event: Event): Result {
  const name = event.name || 'World';

  return {
    message: `Hello ${name}!`,
    language: 'TypeScript'
  };
}

const input: string = process.argv[2] || '{}';
const event: Event = JSON.parse(input);

const result: Result = handler(event);
console.log(JSON.stringify(result));