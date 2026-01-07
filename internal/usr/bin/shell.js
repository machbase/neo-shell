'use strict';

const { ReadLine } = require('readline');
const process = require('process');
const { splitFields } = require('util')
const env = process.env;

const actor = {};
if (!actor.user) {
    actor.user = env.get('NEOSHELL_USER');
    if (!actor.user) {
        actor.user = 'sys';
    }
}
if (!actor.password) {
    actor.password = env.get('NEOSHELL_PASSWORD');
    if (!actor.password) {
        actor.password = 'manager';
    }
}

actor.prompt = (lineno) => {
    return lineno == 0 ? "\x1b[33m" + `${actor.user}` + " \x1b[31mmachbase-neo»\x1b[0m " : "\x1b[31m>\x1b[0m  ";
};

const SQL_VERBS = new Set([
    'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER',
    'TRUNCATE', 'GRANT', 'REVOKE', 'COMMIT', 'ROLLBACK', 'SAVEPOINT',
    'SET', 'SHOW', 'DESCRIBE', 'DESC'
]);

actor.submitOnEnterWhen = (lines, idx) => {
    let maybe = lines.join('').trim().toLowerCase();
    if (maybe === 'exit' || maybe === 'quit') {
        return true;
    }
    if (lines.length == 1 && (maybe == "" || maybe.startsWith('\\'))) {
        return true;
    }
    return lines[idx].endsWith(";");
};

actor.process = (line) => {
    const orgLine = line; // keep original line for history

    line = line.trim(); // trim whitespace
    line = line.replace(/;+\s*$/g, ''); // remove trailing semicolons
    line = line.trim(); // trim whitespace
    if (line.toLowerCase() === 'exit' || line.toLowerCase() === 'quit') {
        process.exit(0);
    }
    else if (line.toLowerCase() === 'clear') {
        console.print('\x1b[2J\x1b[H');
        return;
    }

    if (actor.addHistory) {
        actor.addHistory(orgLine);
    }

    try {
        const fields = splitFields(line);
        if (SQL_VERBS.has(fields[0].toUpperCase())) {
            // handle SQL commands
            process.exec("sql.js", line);
        } else {
            // handle other commands
            if (fields[0] === '\\') {
                // execute jsh shell
                process.exec("/sbin/shell.js", ...fields);
            } 
            else if (fields[0].startsWith('\\')) {
                // execute js command without tailing ';'
                process.exec(fields[0].substring(1), ...fields.slice(1));
            }
            else {
                // execute js that ends with ';'
                process.exec(fields[0], ...fields.slice(1));
            }
        }
    } catch (e) {
        console.println("Process:", e.message);
    }
};

const r = new ReadLine({
    history: 'neo-shell-history',
    prompt: actor.prompt,
    submitOnEnterWhen: actor.submitOnEnterWhen,
});

actor.addHistory = (line) => {
    try {
        r.addHistory(line);
    }catch (e) {
        console.println("AddHistory:", e.message);
    }
};

while (true) {
    try {
        let line = r.readLine();
        if (line instanceof Error) {
            throw line;
        }
        if (line === "" || line === null) {
            continue;
        }
        actor.process(line);
    } catch (e) {
        console.println(e.message);
    }
}
