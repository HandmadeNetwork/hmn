importScripts('../go_wasm_exec.js');

/*
NOTE(ben): The structure here is a little funny but allows for some debouncing. Any postMessages
that got queued up can run all at once, then it can process the latest one.
 */

let ready = false;
let inputData = null;

onmessage = ({ data }) => {
    inputData = data;
    setTimeout(doPreview, 0);
}

const go = new Go();
WebAssembly.instantiateStreaming(fetch('../parsing.wasm'), go.importObject)
    .then(result => {
        go.run(result.instance); // don't await this; we want it to be continuously running
        ready = true;
        setTimeout(doPreview, 0);
    });

const doPreview = () => {
    if (!ready || inputData === null) {
        return;
    }

    const result = parseMarkdown(inputData);
    inputData = null;

    postMessage(result);
}
