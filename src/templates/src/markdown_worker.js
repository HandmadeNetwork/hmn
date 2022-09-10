importScripts('{{ static "go_wasm_exec.js" }}');

// wowee good javascript yeah
const global = Function('return this')();

/*
NOTE(ben): The structure here is a little funny but allows for some debouncing. Any postMessages
that got queued up can run all at once, then it can process the latest one.
 */

let wasmLoaded = false;
let jobs = {};

onmessage = ({ data }) => {
    const { elementID, markdown, parserName } = data;
    jobs[elementID] = { markdown, parserName };
    setTimeout(doPreview, 0);
}

const go = new Go();
WebAssembly.instantiateStreaming(fetch('{{ static "parsing.wasm" }}'), go.importObject)
    .then(result => {
        go.run(result.instance); // don't await this; we want it to be continuously running
        wasmLoaded = true;
        setTimeout(doPreview, 0);
    });

const doPreview = () => {
    if (!wasmLoaded) {
        return;
    }

    for (const [elementID, { markdown, parserName }] of Object.entries(jobs)) {
        const html = global[parserName](markdown);
        postMessage({
            elementID: elementID,
            html: html,
        });
    }
    jobs = {};
}
