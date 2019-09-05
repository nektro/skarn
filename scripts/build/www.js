#!/usr/bin/env node

const fs = require("fs");

fs.readdir("./bin/", function(err, files) {
    //
    const data = `<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <meta http-equiv="X-UA-Compatible" content="ie=edge">
        <title>Skarn Artifacts</title>
        <style>
            ul { font-family: monospace; }
        </style>
    </head>
    <body>
        <h1>Skarn Artifacts</h1>
        <ul>
${files.map(v => v === "index.html" ? "" : `\t\t\t<li><a href="./${v}">${v}</a></li>\n`).join("")}
        </ul>
    </body>
</html>
`;

    //
    fs.writeFile("./bin/index.html", data, function(err) {});
});
