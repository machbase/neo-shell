'use strict';

const process = require('process');
const parseArgs = require('util/parseArgs');
const { splitFields } = require('util');
const { Client } = require('/usr/lib/machcli');
const pretty = require('/usr/lib/pretty');

const options = {
    help: { type: 'boolean', short: 'h', description: 'Show this help message', default: false },
    host: { type: 'string', short: 'H', description: 'Database host', default: '127.0.0.1' },
    port: { type: 'integer', short: 'P', description: 'Database port', default: 5656 },
    user: { type: 'string', description: 'Database user', default: 'sys' },
    password: { type: 'string', description: 'Database password', default: 'manager' },
    output: { type: 'string', short: 'o', description: "output file (default:'-' stdout)", default: '-' },
    compress: { type: 'string', description: "compression type (none, gzip)", default: 'none' },
    timing: { type: 'boolean', short: 'T', description: "print elapsed time", default: false },
    ...pretty.TableArgOptions,
}
const positionals = [
    { name: 'sql', type: 'string', variadic: true, description: 'SQL query to explain' }
];

let showHelp = true;
let config = {};
let args = {};
try {
    const parsed = parseArgs(process.argv.slice(2), {
        options,
        allowPositionals: true,
        allowNegative: true,
        positionals: positionals
    });
    config = parsed.values;
    args = parsed.namedPositionals;
    showHelp = config.help
}
catch (err) {
    console.println(err.message);
}

if (showHelp || (!args.sql) || args.sql.length === 0) {
    console.println(parseArgs.formatHelp({
        usage: 'Usage: sql [options] <sql>',
        options,
        positionals: positionals
    }));
    process.exit(showHelp ? 0 : 1);
}

const sqlText = args.sql.join(' ');
let db, conn, rows;
try {
    db = new Client(config);
    conn = db.connect();
    rows = conn.query(sqlText);

    let box = pretty.Table(config);
    box.appendHeader(rows.columnNames());

    let finalRender = true;
    for (const row of rows) {
        // spread row values
        box.appendRow(box.row(...row));
        if (box.requirePageRender()) {
            // render page
            console.println(box.render());
            // wait for user input to continue if pause is enabled
            if (!box.pauseAndWait()) {
                finalRender = false;
                break;
            }
        }
    }
    // set footer message
    box.setCaption(rows.message)
    // render box
    if (finalRender) {
        console.println(box.render());
    }
} catch (err) {
    console.println("Error: ", err.message);
} finally {
    rows && rows.close();
    conn && conn.close();
    db && db.close();
}
