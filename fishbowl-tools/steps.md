- [ ]  Export with [DiscordChatExporter](https://github.com/Tyrrrz/DiscordChatExporter) CLI 2.34
    
    ```
    DiscordChatExporter.Cli.exe export -c [thread-id] -t [token] -o [fishbowl].html --media
    ```
    
- [ ]  Rename `[fishbowl].html_Files` to `files`, replace links in html
- [ ]  Add target="_blank" to links
    
    ```
    (a href="[^"]+")>
    $1 target="_blank">
    ```
    
- [ ]  Add JQuery to `<head>`

    ```html
    <script src="https://code.jquery.com/jquery-3.6.0.js"></script>
    <script src="https://code.jquery.com/ui/1.13.1/jquery-ui.js"></script>
    <script>
        $( function() {
            $( ".chatlog" ).sortable();
        } );
    </script>
    ```

- [ ]  Drag/delete noise/edit in the browser
- [ ]  Save as `[fishbowl]-dragged.html` (devtools -> `<html>` -> copy outerHTML)
- [ ]  Remove JQuery, sortable classes
    
    ```
    ui-sortable
    ui-sortable-handle
    ```
    
- [ ]  Check `#fishbowl-audience` for highlights
- [ ]  Fix audience avatar paths if anything copied
- [ ]  Fix bad pictures (composite emojis 404)
- [ ]  Fix links with extra braces at the end
    
    ```
    href="[^"]+\)"
    ```
    
- [ ]  Fill in resource links with vscode snippet (select phrase, Ctrl+Shift+P -> Insert Snippet -> hmn-link, paste the url)
- [ ]  Download twemojies

    ```
    go run twemoji.go [fishbowl]-dragged.html files [fishbowl]-twemojied.html
    ```
    
- [ ]  Fix timestamps
    
    ```
    go run timestamps.go [fishbowl]-twemojied.html [fishbowl]-timestamped.html
    ```
    
- [ ]  Create a branch off latest `hmn` master
- [ ]  Create fishbowl folder under `hmn/src/templates/src/fishbowls/`
- [ ]  Copy timestamped html and files, rename html
- [ ]  Remove everything from html but chatlog
- [ ]  Remove js, css and whitney from files
- [ ]  Add content path to `fishbowl.go`
- [ ]  Test locally
- [ ]  Submit a pull request
