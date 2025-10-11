function handler(event) {
  const name = event.name || 'World';

  return `Hello ${name}! This is JavaScript.`;
}

const input = process.argv[2] || '{}';
const event = JSON.parse(input);

const result = handler(event);
console.log(JSON.stringify(result));

