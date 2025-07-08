const { JSDOM } = require('jsdom');
const fs = require('fs');

// The script to run is passed via stdin, as it can be very large.
const scriptToRun = fs.readFileSync(0, 'utf-8');

const dom = new JSDOM('<body></body>', {
    runScripts: "dangerously",
    pretendToBeVisual: true,
});

const window = dom.window;

// Redirect console.log to stdout to capture the answer
window.console.log = (data) => {
    process.stdout.write(String(data));
};

// Handle uncaught exceptions
window.addEventListener('error', (event) => {
    console.error('Script Error:', event.error);
    process.exit(1);
});

try {
    window.eval(scriptToRun);
} catch (e) {
    console.error('Eval Error:', e);
    process.exit(1);
}