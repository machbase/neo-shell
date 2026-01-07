'use strict';

const process = require('process');
const parseArgs = require('util/parseArgs');
const { splitFields } = require('util');
const { Client } = require('/usr/lib/machcli');

const options = {
    help: { type: 'boolean', short: 'h', description: 'Show this help message', default: false },
    host: { type: 'string', short: 'H', description: 'Database host', default: '127.0.0.1' },
    port: { type: 'integer', short: 'P', description: 'Database port', default: 5656 },
    user: { type: 'string', description: 'Database user', default: 'sys' },
    password: { type: 'string', description: 'Database password', default: 'manager' },
    output: { type: 'string', short: 'o', description: "output file (default:'-' stdout)", default: '-' },
    format: { type: 'string', short: 'f', description: "output format (box, csv, json, ndjson)", default: 'box' },
    boxStyle: { type: 'string', description: "box style (simple, bold, double, light, round, colored-bright, colored-dark)", default: 'light' },
    compress: { type: 'string', description: "compression type (none, gzip)", default: 'none' },
    delimiter: { type: 'string', short: 'd', description: "delimiter for csv format", default: ',' },
    rownum: { type: 'boolean', description: "show row numbers", default: true },
    timeformat: { type: 'string', short: 't', description: "time format [ns|ms|s|<timeformat>]", default: 'default' },
    tz: { type: 'string', description: "time zone for handling datetime (default: time zone)", default: 'local' },
    heading: { type: 'boolean', description: "print header", default: true },
    footer: { type: 'boolean', description: "print footer message", default: true },
    pause: { type: 'boolean', description: "pause for the screen paging", default: true },
    timing: { type: 'boolean', short: 'T', description: "print elapsed time", default: false },
    precision: { type: 'integer', short: 'p', description: "set precision of float value to force round", default: -1 },
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

const fields = splitFields(args.sql.join(' '));
const command = fields[0].toLowerCase();

if (command === 'show') {
    show(config, fields.slice(1));
} else if (command === 'describe' || command === 'desc') {
    describe(config, fields[1]);
} else {
    query(config, args.sql.join(' '));
}

function show(config, fields) {
    console.println(`SHOW [${fields.join(' / ')}] command recognized. (not yet implemented)`);
    process.exit(0);
}

function describe(config, tableName) {
    console.println(`DESCRIBE [${tableName}] command recognized. (not yet implemented)`);
    process.exit(0);
}

function query(config, sqlText) {
    const pretty = require('/usr/lib/pretty');

    let db, conn, rows;
    try {
        db = new Client(config);
        conn = db.connect();
        rows = conn.query(sqlText);

        let box = pretty.Table({
            style: config.boxStyle,
            timeformat: config.timeformat,
            tz: config.tz,
            precision: config.precision,
        });
        if(config.rownum) box.setAutoIndex(true);
        box.appendHeader(rows.columnNames());

        // fetch and print rows
        for (const row of rows) {
            // spread row values
            box.appendRow(box.row(...row));
        }
        // render box
        console.println(box.render());
        // print footer message
        console.println(rows.message);
    } catch (err) {
        console.println("Error: ", err.message);
    } finally {
        rows && rows.close();
        conn && conn.close();
        db && db.close();
    }
}
