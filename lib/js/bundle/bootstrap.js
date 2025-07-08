const { JSDOM } = require('jsdom');
const fs = require('fs');

// 1. Get the context from command line and stdin
const pageUrl = process.argv[2];
const scriptToRun = process.argv[3];
const html = fs.readFileSync(0, 'utf-8'); // Read HTML from stdin

if (!pageUrl || !scriptToRun) {
    console.error("Fatal: URL and/or script content not provided.");
    process.exit(1);
}

// 2. THE CORE FIX: Initialize JSDOM with the REAL HTML from the challenge page.
const dom = new JSDOM(html, {
    url: pageUrl,
    runScripts: "dangerously",
    pretendToBeVisual: true,
});

const { window } = dom;
const { document } = window;

// 3. Set a safety timeout
const timeout = setTimeout(() => { process.exit(1); }, 15000);

// 4. Redirect console.log for capturing the answer
window.console.log = (data) => {
    process.stdout.write(String(data));
    clearTimeout(timeout);
    process.exit(0);
};

// 5. Handle errors
window.addEventListener('error', (event) => {
    console.error(event.error ? event.error.stack : event.message);
    clearTimeout(timeout);
    process.exit(1);
});

// 6. Execute the script by creating a script element
const scriptElement = document.createElement('script');
scriptElement.textContent = scriptToRun;
document.body.appendChild(scriptElement);