(function()
{
"use strict";

// for loading more syntax modes at runtime as needed
const codeMirrorModesCDN = "https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.35.0/mode/%%/%%.min.js";

/** @type {CodeMirror} */
let codeMirrorEditor = null;

// preferred default options for CodeMirror editors
let editorOpts           = CodeMirror.defaults;
editorOpts.lineSeparator = "\n";
editorOpts.indentUnit    = 4;
editorOpts.lineNumbers   = true;

/** @type {Element} */
let fileTemplate = null;

/** @type {Element} */
let dirTemplate = null;

/** @type {Object<string, CodeMirror.Doc>} */
let documents = {};

/**
 * the root of our directory tree
 * @type {Element}
 */
let pwd = null;

window.onload = function()
{
    let editorElem = document.querySelector("textarea.editor");

    codeMirrorEditor = CodeMirror.fromTextArea(editorElem, editorOpts);

    pwd = document.querySelector("#pwd");

    pwd.querySelector("button")
       .addEventListener("click", newFileOrDir);

    document.querySelector("#runButton")
            .addEventListener("click", /** @type {EventListener} */ run);

    fileTemplate = document.querySelector("#fileTemplate");
    dirTemplate  = document.querySelector("#dirTemplate");

    let filename = "";
    while(filename === "")
    {
        filename = window.prompt("Enter your filename:");
    }

    newFile(pwd.querySelector("ul"), filename);
};

/**
 * Grabs all file contents, the runCommand, and sends it to the server
 * @param {MouseEvent} event
 */
function run(event)
{
    let progressBar = document.querySelector("#runProgress");
    let outputArea  = document.querySelector("#outputArea");

    let req = new XMLHttpRequest();
    req.open("POST", window.location.href);
    req.setRequestHeader("Content-Type", "application/json");

    req.addEventListener("progress", function(event)
    {
        if(event.lengthComputable)
            progressBar.setAttribute("value", (event.loaded / event.total).toString());
    });

    req.addEventListener("error", () => outputArea.value = "An error occured.");
    req.addEventListener("abort", () => outputArea.value = "Run request was cancelled.");
    req.addEventListener("load", () => outputArea.value = req.responseText);

    let runCmd = {
        cmd:   document.querySelector("#runCommand").value,
        files: walkWorkingDirectory(),
    };

    req.send(JSON.stringify(runCmd));
}

/**
 * switches the CodeMirror editor's current Doc to the Doc related to the file of event.target
 * @param {MouseEvent} event
 */
function switchFile(event)
{
    let file = (/** @type {Element} */ event.currentTarget)
        .parentElement; // li

    codeMirrorEditor.swapDoc(documents[getFullFilename(file)]);
}

/**
 * decides whether to add a file, or a directory based off the event
 * @param {MouseEvent} event
 */
function newFileOrDir(event)
{
    // TODO consider using alertifyjs (https://alertifyjs.com/guide.html) for better new file vs dir UX
    let name = window.prompt("Enter the file (or directory, end with a '/') name:");

    let parent = (/** @type {Element} */ event.currentTarget)
        .parentElement // span
        .parentElement // li
        .querySelector("ul");

    if(name.lastIndexOf("/") === -1)
        newFile(parent, name);
    else
        newDir(parent, name);
}

/**
 * inserts the fileTemplate HTML template into #directoryTree under the directory indicated by event.target
 * @param {Element} parent
 * @param {!string=} filename
 */
function newFile(parent, filename)
{
    (/** @type {Element} */ fileTemplate).content.querySelector("a")
        .textContent = filename;

    let clone = document.importNode((/** @type {Element} */ fileTemplate).content, true);
    parent.appendChild(clone);

    let file = parent.lastElementChild;

    file.querySelector("a")
        .addEventListener("click", /** @type {EventListener} */ switchFile);

    file.querySelector("button.delete")
        .addEventListener("click", /** @type {EventListener} */ delFile);

    let syntaxMode = CodeMirror.findModeByFileName(filename);
    if(syntaxMode)
        loadSyntaxMode(syntaxMode.mode);

    let doc                          = new CodeMirror.Doc("", syntaxMode);
    documents[getFullFilename(file)] = doc;
    codeMirrorEditor.swapDoc(doc);
}

/**
 * creates a new instance of the dirTemplate HTML template for insertion under event.target
 * @param {Element} parent
 * @param {!string=} dirname
 */
function newDir(parent, dirname)
{
    (/** @type {Element} */ dirTemplate).content.querySelector("span")
        .textContent = dirname;

    let clone = document.importNode((/** @type {Element} */ dirTemplate).content, true);
    parent.appendChild(clone);

    let dir = parent.lastElementChild;

    dir.querySelector("button.new")
       .addEventListener("click", /** @type {EventListener} */ newFileOrDir);

    dir.querySelector("button.delete")
       .addEventListener("click", /** @type {EventListener} */ delDir);
}

/**
 * Deletes the file (and corresponding CodeMirror.Doc) indicated by event.target
 * @param {MouseEvent} event
 */
function delFile(event)
{
    let file = (/** @type {Element} */ event.currentTarget)
        .parentElement  // span
        .parentElement; // li
    delete documents[getFullFilename(file)];
    file.parentElement.removeChild(file);
}

/**
 * Deletes the directory (and all children) indicated by event.target
 * @param {MouseEvent} event
 */
function delDir(event)
{
    if(window.confirm("This will delete all files in this directory (cannot be undone)!"))
    {
        // because we have a map of file Elements -> CodeMirror.Docs, one call to removeChild() won't
        // completely clean up after ourselves

        let dir = (/** @type {Element} */ event.currentTarget)
            .parentElement  // span
            .parentElement; // li

        clearDir(dir.querySelector("ul"));

        dir.parentElement.removeChild(dir);
    }
}

/**
 * Recurses down dir and its sub-directories, delete the document for each child file
 * @param {Element} dir
 */
function clearDir(dir)
{
    for(let cn of dir.children)
    {
        // if cn contains an 'a' Element, it's a file

        if(cn.querySelector("a"))
            delete documents[getFullFilename(cn)];
        else
            clearDir(cn.querySelector("ul"));
    }
}

/**
 * @param {!Element=} top If evaluates to false, pwd is used
 * @param {!string=} prefix The filename prefix to prepend to children of top
 * @returns {Array<Object<string>>}
 */
function walkWorkingDirectory(top, prefix)
{
    top    = top || pwd;
    prefix = prefix || "";

    let files = [];

    for(let child of top.querySelector("ul").children)
    {
        // if child contains an 'a' Element, it's a file
        let name = prefix + child.firstElementChild.textContent;

        if(child.querySelector("a"))
            files.push({
                           name: name,
                           data: documents[name].getValue(),
                       });
        else
            files = files.push(...walkWorkingDirectory(child, name));
    }

    return files;
}

/**
 * Given the Element of a file, returns the full filename relative to pwd
 * @param {Element} current
 * @returns {string}
 */
function getFullFilename(current)
{
    let pathArr = [];

    while(current !== pwd)
    {
        pathArr.unshift(current.firstElementChild.textContent);
        current = current
            .parentElement  // ul
            .parentElement; // li
    }

    // don't need to join on path separators - the directories already have one in their name
    return pathArr.join("");
}

/**
 * Adds a script tag to the current document's head with a src for a CDN'd CodeMirror syntax mode
 * (Only does so if the mode hasn't already been loaded)
 * @param {string} mode
 */
function loadSyntaxMode(mode)
{
    let src     = codeMirrorModesCDN.replace(/%%/g, mode);
    let scripts = document.querySelectorAll("script");

    for(let s of scripts)
    {
        // if we already have it, we're done
        if(src === s.getAttribute("src"))
            return;
    }

    // don't already have it, go get it
    let script = document.createElement("script");
    script.setAttribute("type", "text/javascript");
    script.setAttribute("src", src);
    document.getElementsByTagName("head")[0].appendChild(script);
}
}());
